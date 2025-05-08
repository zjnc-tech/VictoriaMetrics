./bin/vmalert \
-rule=apptest/sql/rules.yml \
-datasource.url=http://localhost:5001 \
-datasource.appendTypePrefix \
-notifier.url=http://localhost:9093 \
-remoteWrite.url=http://localhost:5001/vector