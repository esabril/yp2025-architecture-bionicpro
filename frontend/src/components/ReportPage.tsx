import React, { useState } from 'react';
import { useKeycloak } from '@react-keycloak/web';

type ProsthesisReport = {
  user_id: string;
  username: string;
  full_name: string;
  email: string;
  prosthesis_id: string;
  model: string;
  serial_number: string;
  issued_at: string;
  period_start: string;
  period_end: string;
  total_events: number;
  avg_response_time_ms: number;
  max_response_time_ms: number;
  min_battery_level: number;
  avg_signal_quality: number;
  slow_events: number;
  generated_at: string;
};

type ReportsResponse = {
  user_id: string;
  generated_at: string;
  period_start?: string;
  period_end?: string;
  reports: ProsthesisReport[];
};

const ReportPage: React.FC = () => {
  const { keycloak, initialized } = useKeycloak();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [report, setReport] = useState<ReportsResponse | null>(null);
  const [periodStart, setPeriodStart] = useState('');
  const [periodEnd, setPeriodEnd] = useState('');

  const downloadReport = async () => {
    if (!keycloak?.token) {
      setError('Необходимо войти в систему.');
      return;
    }
    if ((periodStart && !periodEnd) || (!periodStart && periodEnd)) {
      setError('Выберите дату начала и дату окончания периода.');
      return;
    }

    try {
      setLoading(true);
      setError(null);

      const params = new URLSearchParams();
      if (periodStart && periodEnd) {
        params.set('period_start', periodStart);
        params.set('period_end', periodEnd);
      }
      const query = params.toString();
      const response = await fetch(`${process.env.REACT_APP_API_URL}/reports${query ? `?${query}` : ''}`, {
        headers: {
          'Authorization': `Bearer ${keycloak.token}`
        }
      });

      const payload = await response.json();
      if (!response.ok) {
        throw new Error(payload.error || 'Не удалось загрузить отчёт.');
      }

      setReport(payload);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Произошла ошибка.');
    } finally {
      setLoading(false);
    }
  };

  if (!initialized) {
    return <div>Loading...</div>;
  }

  if (!keycloak.authenticated) {
    return (
      <div className="flex flex-col items-center justify-center min-h-screen bg-gray-100">
        <button
          onClick={() => keycloak.login()}
          className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"
        >
          Login
        </button>
      </div>
    );
  }

  return (
    <div className="flex flex-col items-center justify-center min-h-screen bg-gray-100">
      <div className="w-full max-w-5xl p-8 bg-white rounded-lg shadow-md">
        <h1 className="text-2xl font-bold mb-6">Usage Reports</h1>

        <div className="flex flex-col gap-3 mb-4 sm:flex-row sm:items-end">
          <label className="flex flex-col text-sm font-medium text-gray-700">
            Period start
            <input
              type="date"
              value={periodStart}
              onChange={(event) => setPeriodStart(event.target.value)}
              className="mt-1 px-3 py-2 border border-gray-300 rounded"
            />
          </label>
          <label className="flex flex-col text-sm font-medium text-gray-700">
            Period end
            <input
              type="date"
              value={periodEnd}
              onChange={(event) => setPeriodEnd(event.target.value)}
              className="mt-1 px-3 py-2 border border-gray-300 rounded"
            />
          </label>
        </div>

        <button
          onClick={downloadReport}
          disabled={loading}
          className={`px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 ${
            loading ? 'opacity-50 cursor-not-allowed' : ''
          }`}
        >
          {loading ? 'Generating Report...' : 'Download Report'}
        </button>

        {error && (
          <div className="mt-4 p-4 bg-red-100 text-red-700 rounded">
            {error}
          </div>
        )}

        {report && (
          <div className="mt-6">
            <div className="mb-4 text-sm text-gray-600">
              User: {report.user_id}. Generated at: {new Date(report.generated_at).toLocaleString()}.
              {report.period_start && report.period_end && (
                <> Period: {report.period_start} - {report.period_end}.</>
              )}
            </div>

            {report.reports.length === 0 ? (
              <div className="p-4 bg-yellow-100 text-yellow-800 rounded">
                Данные отчёта пока недоступны.
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="min-w-full border border-gray-200 text-sm">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="px-3 py-2 text-left border-b">Prosthesis</th>
                      <th className="px-3 py-2 text-left border-b">Model</th>
                      <th className="px-3 py-2 text-right border-b">Events</th>
                      <th className="px-3 py-2 text-right border-b">Avg response</th>
                      <th className="px-3 py-2 text-right border-b">Slow events</th>
                      <th className="px-3 py-2 text-right border-b">Min battery</th>
                      <th className="px-3 py-2 text-right border-b">Signal</th>
                    </tr>
                  </thead>
                  <tbody>
                    {report.reports.map((item) => (
                      <tr key={item.prosthesis_id} className="odd:bg-white even:bg-gray-50">
                        <td className="px-3 py-2 border-b">
                          <div className="font-medium">{item.prosthesis_id}</div>
                          <div className="text-xs text-gray-500">{item.serial_number}</div>
                        </td>
                        <td className="px-3 py-2 border-b">{item.model}</td>
                        <td className="px-3 py-2 text-right border-b">{item.total_events}</td>
                        <td className="px-3 py-2 text-right border-b">
                          {item.avg_response_time_ms.toFixed(1)} ms
                        </td>
                        <td className="px-3 py-2 text-right border-b">{item.slow_events}</td>
                        <td className="px-3 py-2 text-right border-b">{item.min_battery_level}%</td>
                        <td className="px-3 py-2 text-right border-b">
                          {(item.avg_signal_quality * 100).toFixed(1)}%
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};

export default ReportPage;
