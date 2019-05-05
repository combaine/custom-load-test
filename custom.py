#!/usr/bin/env python3
"""Aggregator with extensions support"""

import importlib
import logging
import os
import time
from concurrent import futures

import grpc
import msgpack

import rpc_pb2
import rpc_pb2_grpc


class Custom(rpc_pb2_grpc.CustomAggregatorServicer):
    """Combaine custom plugin loader"""

    def __init__(self):
        self.log = logging.getLogger("combaine")

        self.path = os.environ.get('PLUGINS_PATH', '/usr/lib/yandex/combaine/custom')
        self.all_custom_parsers = self.load_plugins()

    def load_plugins(self):
        parsers = {}
        names = set(c.split('.')[0] for c in os.listdir(self.path) if self._is_plugin(c))
        for name in names:
            plugin_file = self.get_plugin_file(name)
            if plugin_file is None:
                self.log.debug("load_plugins skip: %s", name)
                continue

            try:
                spec = importlib.util.spec_from_file_location(name, plugin_file)
                module = importlib.util.module_from_spec(spec)
                spec.loader.exec_module(module)
                self.log.debug("Import parsers from: %s", plugin_file)
                for item in (x for x in dir(module) if self._is_candidate(x)):
                    candidate = getattr(module, item)
                    if callable(candidate):
                        parsers[item] = candidate
            except Exception as err:
                self.log.error("ImportError. Module: %s %s", name, repr(err))

        self.log.info("%s are available custom plugin for parsing", parsers.keys())
        return parsers

    def get_plugin_file(self, name):
        file_base = os.path.join(self.path, name)
        for ext in importlib.machinery.EXTENSION_SUFFIXES:
            mod_name = file_base + ext
            if os.path.exists(mod_name):
                return mod_name
        # try compiled file
        mod_name = file_base + '.py'
        mod_cache = importlib.util.cache_from_source(mod_name)
        if os.path.exists(mod_cache):
            return mod_cache
        if os.path.exists(mod_name):
            return mod_name
        return None

    @staticmethod
    def _is_plugin(name):
        maybe = any(name.endswith(e) for e in importlib.machinery.all_suffixes())
        return not name.startswith("_") and maybe

    @staticmethod
    def _is_candidate(name):
        return not name.startswith("_") and name[0].isupper()

    def GetClass(self, name, context):
        klass = self.all_custom_parsers.get(name, None)
        if not klass:
            context.set_code(grpc.StatusCode.NOT_FOUND)
            msg = "Class '{}' not found!".format(klass)
            context.set_details(msg)
            self.log.error(msg)
            raise NameError(msg)
        return klass

    def GetConfig(self, request):
        cfg = msgpack.unpackb(request.task.config, raw=False)
        logger = logging.LoggerAdapter(self.log, {'tid': request.task.id})
        cfg['logger'] = logger
        return cfg

    def Ping(self, request, context):
        return rpc_pb2.PongResponse()

    def AggregateHost(self, request, context):
        """
        Gets the result of a single host,
        performs parsing and their aggregation
        """
        cfg = self.GetConfig(request)

        klass = self.GetClass(request.class_name, context)

        prevtime = request.task.frame.previous
        currtime = request.task.frame.current
        hostname = request.task.meta.get("host")
        result = klass(cfg).aggregate_host(request.payload, prevtime, currtime, hostname)

        if cfg.get("logHostResult", False):
            self.log.info("Aggregate host result %s: %s", request.task.meta, result)

        result_bytes = msgpack.packb(result)
        return rpc_pb2.AggregateHostResponse(result=result_bytes)

    def AggregateGroup(self, request, context):
        """
        Receives a list of results from the aggregate_host,
        and performs aggregation by group
        """
        payload = [msgpack.unpackb(i) for i in request.payload]
        cfg = self.GetConfig(request)
        logger = cfg['logger']

        klass = self.GetClass(request.class_name, context)
        result = klass(cfg).aggregate_group(payload)

        if cfg.get("logGroupResult", False):
            logger.info("Aggregate group result %s: %s", request.task.meta, result)
        result_bytes = msgpack.packb(result)
        return rpc_pb2.AggregateGroupResponse(result=result_bytes)


def serve():
    server = grpc.server(
        futures.ThreadPoolExecutor(max_workers=1),
        # grpc.max_send_message_length grpc.max_receive_message_length
        options=[('grpc.max_send_message_length', 128 * 1024 * 1024),
                 ('grpc.max_receive_message_length', 128 * 1024 * 1024)])
    rpc_pb2_grpc.add_CustomAggregatorServicer_to_server(Custom(), server)
    server.add_insecure_port('[::]:50051')
    server.start()
    try:
        while True:
            time.sleep(60 * 60 * 24)
    except KeyboardInterrupt:
        server.stop(0)


if __name__ == '__main__':
    logging.basicConfig()
    logging.getLogger().setLevel(logging.DEBUG)
    serve()
