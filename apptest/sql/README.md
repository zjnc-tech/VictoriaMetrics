# Preparation

make vmalert

# Start alertmanager

git clone https://github.com/prometheus/alertmanager.git
cd path/to/alertmanager
make build
cp path/to/VictoriaMetrics/apptest/sql/alertmanager.yml ./
./alertmanager --config.file=alertmanager.yml

# Start datasource, webhook, remote write server

go run apptest/sql/thirdservice.go

# Start vmalert

./apptest/sql/vmalert.sh
