package reports

// DashboardHTML returns the HTML for the performance dashboard.
func DashboardHTML() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>kallm - Cache Performance Dashboard</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            background: #0f172a;
            color: #e2e8f0;
            min-height: 100vh;
        }
        .header {
            background: linear-gradient(135deg, #1e293b 0%, #0f172a 100%);
            padding: 1.5rem 2rem;
            border-bottom: 1px solid #334155;
        }
        .header h1 {
            font-size: 1.5rem;
            font-weight: 600;
            color: #f8fafc;
        }
        .header p { color: #94a3b8; font-size: 0.875rem; margin-top: 0.25rem; }
        .container { padding: 1.5rem 2rem; max-width: 1400px; margin: 0 auto; }

        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 1rem;
            margin-bottom: 1.5rem;
        }
        .stat-card {
            background: #1e293b;
            border-radius: 0.75rem;
            padding: 1.25rem;
            border: 1px solid #334155;
        }
        .stat-label { color: #94a3b8; font-size: 0.75rem; text-transform: uppercase; letter-spacing: 0.05em; }
        .stat-value { font-size: 1.75rem; font-weight: 700; color: #f8fafc; margin-top: 0.25rem; }
        .stat-value.green { color: #4ade80; }
        .stat-value.blue { color: #60a5fa; }
        .stat-value.purple { color: #a78bfa; }
        .stat-value.yellow { color: #facc15; }

        .charts-grid {
            display: grid;
            grid-template-columns: repeat(2, 1fr);
            gap: 1rem;
            margin-bottom: 1.5rem;
        }
        @media (max-width: 1024px) { .charts-grid { grid-template-columns: 1fr; } }

        .chart-card {
            background: #1e293b;
            border-radius: 0.75rem;
            padding: 1.25rem;
            border: 1px solid #334155;
        }
        .chart-card h3 { font-size: 0.875rem; color: #f8fafc; margin-bottom: 1rem; }
        .chart-container { position: relative; height: 200px; }

        .table-card {
            background: #1e293b;
            border-radius: 0.75rem;
            padding: 1.25rem;
            border: 1px solid #334155;
        }
        .table-card h3 { font-size: 0.875rem; color: #f8fafc; margin-bottom: 1rem; }
        table { width: 100%; border-collapse: collapse; font-size: 0.8rem; }
        th { text-align: left; color: #94a3b8; font-weight: 500; padding: 0.5rem; border-bottom: 1px solid #334155; }
        td { padding: 0.5rem; border-bottom: 1px solid #1e293b; }
        .badge {
            display: inline-block;
            padding: 0.125rem 0.5rem;
            border-radius: 9999px;
            font-size: 0.7rem;
            font-weight: 500;
        }
        .badge.hit { background: #166534; color: #4ade80; }
        .badge.miss { background: #7f1d1d; color: #fca5a5; }

        .refresh-info {
            text-align: center;
            color: #64748b;
            font-size: 0.75rem;
            margin-top: 1rem;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>kallm Cache Performance</h1>
        <p>Real-time semantic cache metrics and analytics</p>
    </div>

    <div class="container">
        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-label">Hit Rate</div>
                <div class="stat-value green" id="hitRate">--%</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Total Requests</div>
                <div class="stat-value blue" id="totalRequests">--</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Avg Latency</div>
                <div class="stat-value purple" id="avgLatency">--ms</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Est. Savings</div>
                <div class="stat-value yellow" id="savings">$--</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Cache Hits</div>
                <div class="stat-value" id="cacheHits">--</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Cache Misses</div>
                <div class="stat-value" id="cacheMisses">--</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Requests/min</div>
                <div class="stat-value" id="reqPerMin">--</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Uptime</div>
                <div class="stat-value" id="uptime">--</div>
            </div>
        </div>

        <div class="charts-grid">
            <div class="chart-card">
                <h3>Hit Rate Over Time (%)</h3>
                <div class="chart-container"><canvas id="hitRateChart"></canvas></div>
            </div>
            <div class="chart-card">
                <h3>Latency Over Time (ms)</h3>
                <div class="chart-container"><canvas id="latencyChart"></canvas></div>
            </div>
            <div class="chart-card">
                <h3>Latency Distribution</h3>
                <div class="chart-container"><canvas id="latencyDistChart"></canvas></div>
            </div>
            <div class="chart-card">
                <h3>Similarity Distribution (Cache Hits)</h3>
                <div class="chart-container"><canvas id="similarityDistChart"></canvas></div>
            </div>
        </div>

        <div class="table-card">
            <h3>Recent Requests</h3>
            <table>
                <thead>
                    <tr>
                        <th>Time</th>
                        <th>Status</th>
                        <th>Similarity</th>
                        <th>Latency</th>
                    </tr>
                </thead>
                <tbody id="requestsTable"></tbody>
            </table>
        </div>

        <div class="refresh-info">Auto-refreshes every 5 seconds</div>
    </div>

    <script>
        const chartOptions = {
            responsive: true,
            maintainAspectRatio: false,
            plugins: { legend: { display: false } },
            scales: {
                x: { grid: { color: '#334155' }, ticks: { color: '#94a3b8', maxTicksLimit: 6 } },
                y: { grid: { color: '#334155' }, ticks: { color: '#94a3b8' } }
            }
        };

        const hitRateChart = new Chart(document.getElementById('hitRateChart'), {
            type: 'line',
            data: { labels: [], datasets: [{ data: [], borderColor: '#4ade80', backgroundColor: 'rgba(74, 222, 128, 0.1)', fill: true, tension: 0.3 }] },
            options: { ...chartOptions, scales: { ...chartOptions.scales, y: { ...chartOptions.scales.y, min: 0, max: 100 } } }
        });

        const latencyChart = new Chart(document.getElementById('latencyChart'), {
            type: 'line',
            data: { labels: [], datasets: [{ data: [], borderColor: '#a78bfa', backgroundColor: 'rgba(167, 139, 250, 0.1)', fill: true, tension: 0.3 }] },
            options: chartOptions
        });

        const latencyDistChart = new Chart(document.getElementById('latencyDistChart'), {
            type: 'bar',
            data: { labels: [], datasets: [{ data: [], backgroundColor: ['#4ade80', '#60a5fa', '#a78bfa', '#facc15', '#f87171'] }] },
            options: { ...chartOptions, scales: { ...chartOptions.scales, y: { ...chartOptions.scales.y, beginAtZero: true } } }
        });

        const similarityDistChart = new Chart(document.getElementById('similarityDistChart'), {
            type: 'doughnut',
            data: { labels: [], datasets: [{ data: [], backgroundColor: ['#4ade80', '#60a5fa', '#a78bfa', '#facc15', '#f87171'] }] },
            options: { responsive: true, maintainAspectRatio: false, plugins: { legend: { position: 'right', labels: { color: '#94a3b8' } } } }
        });

        function formatTime(ts) {
            return new Date(ts).toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' });
        }

        async function fetchData() {
            try {
                const resp = await fetch('/reports/data');
                const data = await resp.json();

                // Update stats
                document.getElementById('hitRate').textContent = data.hit_rate.toFixed(1) + '%';
                document.getElementById('totalRequests').textContent = data.total_requests.toLocaleString();
                document.getElementById('avgLatency').textContent = data.avg_latency_ms.toFixed(1) + 'ms';
                document.getElementById('savings').textContent = '$' + data.total_savings_usd.toFixed(4);
                document.getElementById('cacheHits').textContent = data.total_hits.toLocaleString();
                document.getElementById('cacheMisses').textContent = data.total_misses.toLocaleString();
                document.getElementById('reqPerMin').textContent = data.requests_per_min.toFixed(1);
                document.getElementById('uptime').textContent = data.uptime;

                // Update hit rate chart
                if (data.hit_rate_history && data.hit_rate_history.length > 0) {
                    hitRateChart.data.labels = data.hit_rate_history.map(p => formatTime(p.timestamp));
                    hitRateChart.data.datasets[0].data = data.hit_rate_history.map(p => p.value);
                    hitRateChart.update('none');
                }

                // Update latency chart
                if (data.latency_history && data.latency_history.length > 0) {
                    latencyChart.data.labels = data.latency_history.map(p => formatTime(p.timestamp));
                    latencyChart.data.datasets[0].data = data.latency_history.map(p => p.value);
                    latencyChart.update('none');
                }

                // Update latency distribution
                if (data.latency_distribution) {
                    latencyDistChart.data.labels = data.latency_distribution.map(b => b.bucket);
                    latencyDistChart.data.datasets[0].data = data.latency_distribution.map(b => b.count);
                    latencyDistChart.update('none');
                }

                // Update similarity distribution
                if (data.similarity_distribution) {
                    similarityDistChart.data.labels = data.similarity_distribution.map(b => b.bucket);
                    similarityDistChart.data.datasets[0].data = data.similarity_distribution.map(b => b.count);
                    similarityDistChart.update('none');
                }

                // Update recent requests table
                const tbody = document.getElementById('requestsTable');
                tbody.innerHTML = '';
                if (data.recent_requests) {
                    data.recent_requests.slice(0, 20).forEach(req => {
                        const tr = document.createElement('tr');
                        tr.innerHTML = ` + "`" + `
                            <td>${formatTime(req.timestamp)}</td>
                            <td><span class="badge ${req.cache_hit ? 'hit' : 'miss'}">${req.cache_hit ? 'HIT' : 'MISS'}</span></td>
                            <td>${req.cache_hit ? (req.similarity * 100).toFixed(2) + '%' : '-'}</td>
                            <td>${req.latency_ms}ms</td>
                        ` + "`" + `;
                        tbody.appendChild(tr);
                    });
                }
            } catch (e) {
                console.error('Failed to fetch data:', e);
            }
        }

        fetchData();
        setInterval(fetchData, 5000);
    </script>
</body>
</html>`
}
