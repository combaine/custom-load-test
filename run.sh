PLUGINS_PATH=./custom/ python3 ./custom.py &
background=$!

trap 'pkill -f "python3 ./custom.py"; exit 0' INT
until go run cmd/charge/main.go cmd/charge/main_gen.go; do
    sleep 1
done
