compose-up:
	docker compose up -d

compose-total-down:
	docker compose down --rmi local -v

etl:
	docker compose exec -T airflow airflow tasks test prepare_user_reports_mart create_reporting_table 2026-05-17T10:00:00+00:00
	docker compose exec -T airflow airflow tasks test prepare_user_reports_mart extract_and_load 2026-05-17T10:00:00+00:00

ch:
	docker compose exec -T clickhouse clickhouse-client --user default --password clickhouse_password --query "SELECT count() FROM user_reports"
	docker compose exec -T clickhouse clickhouse-client --user default --password clickhouse_password --query "SELECT * FROM user_reports FORMAT Vertical"