from __future__ import annotations

import json
from datetime import datetime, timedelta

import pendulum
import requests
from airflow.decorators import dag, task
from airflow.providers.postgres.hooks.postgres import PostgresHook


CLICKHOUSE_URL = "http://clickhouse:8123"
CLICKHOUSE_AUTH = ("default", "clickhouse_password")


def clickhouse_query(query: str, data: str | None = None) -> None:
    if data:
        response = requests.post(
            CLICKHOUSE_URL,
            data=(query.strip() + "\n" + data).encode("utf-8"),
            auth=CLICKHOUSE_AUTH,
            timeout=30,
        )
    else:
        response = requests.post(
            CLICKHOUSE_URL,
            params={"query": query},
            auth=CLICKHOUSE_AUTH,
            timeout=30,
        )
    if response.status_code >= 300:
        raise RuntimeError(f"ClickHouse query failed: {response.status_code} {response.text}")


@dag(
    dag_id="prepare_user_reports_mart",
    description="Builds a ClickHouse reporting mart from CRM and telemetry data.",
    start_date=pendulum.datetime(2026, 1, 1, tz="UTC"),
    schedule="*/30 * * * *",
    catchup=False,
    max_active_runs=1,
    default_args={"retries": 2, "retry_delay": timedelta(minutes=1)},
    tags=["bionicpro", "reports", "etl"],
)
def prepare_user_reports_mart():
    @task
    def create_reporting_table() -> None:
        clickhouse_query(
            """
            CREATE TABLE IF NOT EXISTS user_reports (
                user_id String,
                keycloak_username String,
                full_name String,
                email String,
                prosthesis_id String,
                model String,
                serial_number String,
                issued_at Date,
                period_start DateTime,
                period_end DateTime,
                total_events UInt64,
                avg_response_time_ms Float64,
                max_response_time_ms UInt32,
                min_battery_level UInt8,
                avg_signal_quality Float64,
                slow_events UInt64,
                generated_at DateTime
            )
            ENGINE = ReplacingMergeTree(generated_at)
            PARTITION BY toYYYYMM(period_end)
            ORDER BY (keycloak_username, prosthesis_id)
            """
        )

    @task
    def extract_and_load() -> int:
        crm = PostgresHook(postgres_conn_id="crm_db")
        telemetry = PostgresHook(postgres_conn_id="telemetry_db")

        customers = crm.get_records(
            """
            SELECT c.user_id,
                   c.keycloak_username,
                   c.full_name,
                   c.email,
                   p.prosthesis_id,
                   p.model,
                   p.serial_number,
                   p.issued_at
            FROM customers c
            JOIN prostheses p ON p.user_id = c.user_id
            """
        )

        telemetry_rows = telemetry.get_records(
            """
            SELECT prosthesis_id,
                   min(event_time) AS period_start,
                   max(event_time) AS period_end,
                   count(*) AS total_events,
                   avg(response_time_ms) AS avg_response_time_ms,
                   max(response_time_ms) AS max_response_time_ms,
                   min(battery_level) AS min_battery_level,
                   avg(signal_quality) AS avg_signal_quality,
                   count(*) FILTER (WHERE response_time_ms > 100) AS slow_events
            FROM telemetry_events
            GROUP BY prosthesis_id
            """
        )

        telemetry_by_prosthesis = {row[0]: row[1:] for row in telemetry_rows}
        generated_at = datetime.utcnow().replace(microsecond=0).isoformat(sep=" ")

        rows = []
        loaded = 0

        for customer in customers:
            (
                user_id,
                keycloak_username,
                full_name,
                email,
                prosthesis_id,
                model,
                serial_number,
                issued_at,
            ) = customer

            metrics = telemetry_by_prosthesis.get(prosthesis_id)
            if not metrics:
                continue

            (
                period_start,
                period_end,
                total_events,
                avg_response_time_ms,
                max_response_time_ms,
                min_battery_level,
                avg_signal_quality,
                slow_events,
            ) = metrics

            rows.append(
                json.dumps(
                    {
                        "user_id": user_id,
                        "keycloak_username": keycloak_username,
                        "full_name": full_name,
                        "email": email,
                        "prosthesis_id": prosthesis_id,
                        "model": model,
                        "serial_number": serial_number,
                        "issued_at": issued_at.isoformat(),
                        "period_start": period_start.replace(tzinfo=None, microsecond=0).isoformat(sep=" "),
                        "period_end": period_end.replace(tzinfo=None, microsecond=0).isoformat(sep=" "),
                        "total_events": total_events,
                        "avg_response_time_ms": float(avg_response_time_ms),
                        "max_response_time_ms": max_response_time_ms,
                        "min_battery_level": min_battery_level,
                        "avg_signal_quality": float(avg_signal_quality),
                        "slow_events": slow_events,
                        "generated_at": generated_at,
                    },
                    ensure_ascii=False,
                )
            )
            loaded += 1

        if loaded == 0:
            return 0

        clickhouse_query("TRUNCATE TABLE user_reports")
        clickhouse_query(
            """
            INSERT INTO user_reports FORMAT JSONEachRow
            """,
            "\n".join(rows),
        )
        return loaded

    create_reporting_table() >> extract_and_load()


prepare_user_reports_mart()
