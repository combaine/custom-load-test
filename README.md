Test throughput of the custom.py on grpc with multiprocessing
=============================================================

Install grpc from source
```bash
sudo -H pip3 install grpcio --no-binary grpcio
sudo -H pip3 install grpcio-tools
```
Install msgp
```bash
GO111MODULE=on go get -u -t github.com/tinylib/msgp
```

Push the Fire button
```bash
sudo -H pip3 install Cython
./gen_code.sh
./run.sh
```
Pause/Resume loading by pressing space key
Stop by pressing Ctrl+C
