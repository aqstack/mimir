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
            padding: 1.5rem 2.5rem;
            border-bottom: 1px solid #334155;
        }
        .header h1 {
            font-size: 1.5rem;
            font-weight: 600;
            color: #f8fafc;
        }
        .header p { color: #94a3b8; font-size: 0.875rem; margin-top: 0.25rem; }
        .container { padding: 2rem 2.5rem; max-width: 1400px; margin: 0 auto; }

        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
            gap: 1.25rem;
            margin-bottom: 2rem;
        }
        .stat-card {
            background: #1e293b;
            border-radius: 0.75rem;
            padding: 1.5rem;
            border: 1px solid #334155;
        }
        .stat-label { color: #94a3b8; font-size: 0.75rem; text-transform: uppercase; letter-spacing: 0.05em; font-weight: 500; }
        .stat-value { font-size: 1.75rem; font-weight: 700; color: #f8fafc; margin-top: 0.5rem; }
        .stat-value.green { color: #4ade80; }
        .stat-value.blue { color: #60a5fa; }
        .stat-value.purple { color: #a78bfa; }
        .stat-value.yellow { color: #facc15; }

        .charts-grid {
            display: grid;
            grid-template-columns: repeat(2, 1fr);
            gap: 1.25rem;
            margin-bottom: 2rem;
        }
        @media (max-width: 1024px) { .charts-grid { grid-template-columns: 1fr; } }

        .chart-card {
            background: #1e293b;
            border-radius: 0.75rem;
            padding: 1.5rem;
            border: 1px solid #334155;
        }
        .chart-card h3 { font-size: 0.875rem; color: #f8fafc; margin-bottom: 1rem; font-weight: 600; }
        .chart-container { position: relative; height: 200px; }

        .table-card {
            background: #1e293b;
            border-radius: 0.75rem;
            padding: 1.5rem;
            border: 1px solid #334155;
            margin-bottom: 2rem;
        }
        .table-card h3 { font-size: 0.875rem; color: #f8fafc; margin-bottom: 1rem; font-weight: 600; }
        table { width: 100%; border-collapse: collapse; font-size: 0.8rem; }
        th { text-align: left; color: #94a3b8; font-weight: 500; padding: 0.75rem; border-bottom: 1px solid #334155; }
        td { padding: 0.75rem; border-bottom: 1px solid #1e293b; color: #e2e8f0; }
        tr:hover { background: #334155; }
        .badge {
            display: inline-block;
            padding: 0.25rem 0.625rem;
            border-radius: 9999px;
            font-size: 0.7rem;
            font-weight: 600;
        }
        .badge.hit { background: #166534; color: #4ade80; }
        .badge.miss { background: #7f1d1d; color: #fca5a5; }

        .refresh-info {
            text-align: center;
            color: #64748b;
            font-size: 0.75rem;
            margin-top: 1.5rem;
        }

        .test-panel { min-height: 200px; }
        .test-form { display: flex; flex-direction: column; gap: 1rem; }
        .test-form textarea {
            background: #0f172a;
            border: 1px solid #334155;
            border-radius: 0.5rem;
            color: #e2e8f0;
            padding: 1rem;
            font-size: 0.875rem;
            resize: vertical;
            min-height: 70px;
            font-family: inherit;
        }
        .test-form textarea:focus { outline: none; border-color: #60a5fa; }
        .test-controls { display: flex; gap: 0.75rem; }
        .test-controls select, .test-controls button {
            padding: 0.625rem 1.25rem;
            border-radius: 0.5rem;
            font-size: 0.875rem;
            cursor: pointer;
        }
        .test-controls select {
            background: #0f172a;
            border: 1px solid #334155;
            color: #e2e8f0;
            flex: 1;
        }
        .test-controls button, .traffic-presets button {
            background: #3b82f6;
            border: none;
            color: white;
            font-weight: 500;
            transition: all 0.2s;
        }
        .test-controls button:hover, .traffic-presets button:hover { background: #2563eb; }
        .test-controls button:disabled, .traffic-presets button:disabled {
            background: #475569;
            cursor: not-allowed;
        }
        .test-result {
            background: #0f172a;
            border-radius: 0.5rem;
            padding: 1rem;
            font-size: 0.8rem;
            font-family: 'SF Mono', Monaco, monospace;
            max-height: 120px;
            overflow-y: auto;
            white-space: pre-wrap;
            word-break: break-word;
            color: #e2e8f0;
        }
        .test-result.hit { border-left: 3px solid #4ade80; }
        .test-result.miss { border-left: 3px solid #f87171; }
        .test-result.error { border-left: 3px solid #facc15; color: #facc15; }

        .traffic-options { display: flex; gap: 1.5rem; }
        .traffic-options label {
            display: flex;
            align-items: center;
            gap: 0.5rem;
            font-size: 0.8rem;
            color: #94a3b8;
        }
        .traffic-options input {
            width: 70px;
            padding: 0.5rem;
            background: #0f172a;
            border: 1px solid #334155;
            border-radius: 0.375rem;
            color: #e2e8f0;
            font-size: 0.8rem;
        }
        .traffic-presets { display: flex; gap: 0.5rem; flex-wrap: wrap; }
        .traffic-presets button { padding: 0.5rem 1rem; font-size: 0.75rem; border-radius: 0.375rem; }
        .progress-bar {
            height: 6px;
            background: #334155;
            border-radius: 3px;
            overflow: hidden;
        }
        .progress-bar > div {
            height: 100%;
            background: linear-gradient(90deg, #22c55e, #4ade80);
            width: 0%;
            transition: width 0.3s;
        }

        .logs-panel { margin-bottom: 0; }
        .logs-container {
            background: #0f172a;
            border-radius: 0.5rem;
            padding: 1rem;
            font-family: 'SF Mono', Monaco, Menlo, monospace;
            font-size: 0.75rem;
            height: 200px;
            overflow-y: auto;
            line-height: 1.5;
            color: #e2e8f0;
        }
        .log-line { margin: 3px 0; }
        .log-line.hit { color: #4ade80; }
        .log-line.miss { color: #f87171; }
        .log-line.info { color: #94a3b8; }
        .log-line.error { color: #fbbf24; }

        .clear-btn {
            float: right;
            padding: 4px 12px;
            font-size: 0.7rem;
            background: #334155;
            border: 1px solid #475569;
            color: #e2e8f0;
            border-radius: 4px;
            cursor: pointer;
            transition: all 0.2s;
        }
        .clear-btn:hover { background: #475569; }
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
            <div class="chart-card test-panel">
                <h3>Test Prompt</h3>
                <div class="test-form">
                    <textarea id="testPrompt" placeholder="Enter a prompt to test caching...">What is 2+2?</textarea>
                    <div class="test-controls">
                        <select id="testModel">
                            <option value="llama3.2:1b">llama3.2:1b</option>
                            <option value="gpt-4">gpt-4</option>
                            <option value="gpt-3.5-turbo">gpt-3.5-turbo</option>
                        </select>
                        <button id="sendBtn" onclick="sendTestPrompt()">Send</button>
                    </div>
                    <div id="testResult" class="test-result"></div>
                </div>
            </div>
            <div class="chart-card test-panel">
                <h3>Traffic Generator</h3>
                <div class="test-form">
                    <div class="traffic-options">
                        <label>Requests: <input type="number" id="trafficCount" value="10" min="1" max="100"></label>
                        <label>Delay (ms): <input type="number" id="trafficDelay" value="100" min="0" max="5000"></label>
                    </div>
                    <div class="traffic-presets">
                        <button onclick="generateTraffic('identical')" title="Same query repeated - 100% cache hits expected">Identical</button>
                        <button onclick="generateTraffic('similar')" title="Semantically similar queries - high cache hit rate expected">Similar</button>
                        <button onclick="generateTraffic('coding')" title="Programming questions with variations">Coding</button>
                        <button onclick="generateTraffic('devops')" title="DevOps/infrastructure questions">DevOps</button>
                        <button onclick="generateTraffic('random')" title="Diverse topics - mostly cache misses expected">Random</button>
                    </div>
                    <div id="trafficStatus" class="test-result"></div>
                    <div class="progress-bar"><div id="trafficProgress"></div></div>
                </div>
            </div>
        </div>

        <div class="table-card logs-panel">
            <h3>Live Logs <button onclick="clearLogs()" class="clear-btn">Clear</button></h3>
            <div id="logsContainer" class="logs-container"></div>
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
                        <th>Prompt</th>
                    </tr>
                </thead>
                <tbody id="requestsTable"></tbody>
            </table>
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
            data: { labels: [], datasets: [{ data: [], borderColor: '#4ade80', backgroundColor: 'rgba(74, 222, 128, 0.1)', fill: true, tension: 0.3, borderWidth: 2 }] },
            options: { ...chartOptions, scales: { ...chartOptions.scales, y: { ...chartOptions.scales.y, min: 0, max: 100 } } }
        });

        const latencyChart = new Chart(document.getElementById('latencyChart'), {
            type: 'line',
            data: { labels: [], datasets: [{ data: [], borderColor: '#a78bfa', backgroundColor: 'rgba(167, 139, 250, 0.1)', fill: true, tension: 0.3, borderWidth: 2 }] },
            options: chartOptions
        });

        const latencyDistChart = new Chart(document.getElementById('latencyDistChart'), {
            type: 'bar',
            data: { labels: [], datasets: [{ data: [], backgroundColor: ['#4ade80', '#60a5fa', '#a78bfa', '#facc15', '#f87171'], borderRadius: 4 }] },
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
                        const prompt = req.prompt ? req.prompt.replace(/\n/g, ' ') : '-';
                        tr.innerHTML = ` + "`" + `
                            <td style="white-space:nowrap">${formatTime(req.timestamp)}</td>
                            <td><span class="badge ${req.cache_hit ? 'hit' : 'miss'}">${req.cache_hit ? 'HIT' : 'MISS'}</span></td>
                            <td style="white-space:nowrap">${req.cache_hit ? (req.similarity * 100).toFixed(2) + '%' : '-'}</td>
                            <td style="white-space:nowrap">${req.latency_ms}ms</td>
                            <td style="word-break:break-word">${prompt}</td>
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

        // Test prompt functionality
        async function sendTestPrompt() {
            const btn = document.getElementById('sendBtn');
            const result = document.getElementById('testResult');
            const prompt = document.getElementById('testPrompt').value;
            const model = document.getElementById('testModel').value;

            btn.disabled = true;
            btn.textContent = 'Sending...';
            result.className = 'test-result';
            result.textContent = 'Sending request...';

            try {
                const start = performance.now();
                const resp = await fetch('/v1/chat/completions', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        model: model,
                        messages: [{ role: 'user', content: prompt }]
                    })
                });

                const latency = Math.round(performance.now() - start);
                const cacheStatus = resp.headers.get('X-Kallm-Cache') || 'MISS';
                const similarity = resp.headers.get('X-Kallm-Similarity') || '-';
                const data = await resp.json();

                const content = data.choices?.[0]?.message?.content || JSON.stringify(data);
                result.className = 'test-result ' + cacheStatus.toLowerCase();
                result.textContent = ` + "`" + `[${cacheStatus}] ${latency}ms ${cacheStatus === 'HIT' ? '(sim: ' + similarity + ')' : ''}
${content}` + "`" + `;

                fetchData(); // Refresh stats
            } catch (e) {
                result.className = 'test-result error';
                result.textContent = 'Error: ' + e.message;
            } finally {
                btn.disabled = false;
                btn.textContent = 'Send';
            }
        }

        // Traffic generator
        const trafficPrompts = {
            identical: ['Explain the difference between SQL and NoSQL databases'],
            similar: [
                // Database questions - should have high semantic similarity
                'Explain the difference between SQL and NoSQL databases',
                'What are the key differences between SQL and NoSQL?',
                'Compare SQL databases to NoSQL databases',
                'SQL vs NoSQL - what is the difference?',
                'How do relational databases differ from NoSQL databases?',
                // Python questions - should have high semantic similarity
                'How do I read a file in Python?',
                'What is the Python code to read a file?',
                'Show me how to open and read a file in Python',
                'Python file reading example',
                // API questions
                'What is a REST API?',
                'Explain REST APIs',
                'What does REST API mean?',
                'How do REST APIs work?'
            ],
            random: [
                'Explain the difference between TCP and UDP protocols',
                'What is the time complexity of quicksort?',
                'How does garbage collection work in Java?',
                'Explain the CAP theorem in distributed systems',
                'What is the difference between process and thread?',
                'How does HTTPS encryption work?',
                'Explain microservices architecture',
                'What is Docker and how does containerization work?',
                'Explain the concept of eventual consistency',
                'What is a load balancer and how does it work?',
                'Describe the differences between REST and GraphQL',
                'How does DNS resolution work?',
                'What is the purpose of an index in a database?',
                'Explain OAuth 2.0 authentication flow',
                'What is the difference between horizontal and vertical scaling?',
                'How do WebSockets differ from HTTP?',
                'Explain the concept of database sharding',
                'What is a reverse proxy?',
                'How does Redis caching work?',
                'Explain the publish-subscribe pattern'
            ],
            coding: [
                'Write a function to reverse a string in Python',
                'How do I reverse a string in Python?',
                'Python code to reverse a string',
                'Show me string reversal in Python',
                'Implement a function to check if a number is prime',
                'Write code to check for prime numbers',
                'How to determine if a number is prime?',
                'Prime number checking algorithm',
                'How do I sort a list in Python?',
                'Python list sorting methods',
                'Sort a list in ascending order Python',
                'What is the best way to sort lists in Python?'
            ],
            devops: [
                'How do I create a Kubernetes deployment?',
                'Kubernetes deployment YAML example',
                'Create a deployment in K8s',
                'Write a Kubernetes deployment manifest',
                'How to set up a CI/CD pipeline?',
                'Explain CI/CD pipeline setup',
                'What are the steps to create a CI/CD pipeline?',
                'CI/CD best practices',
                'How do I write a Dockerfile?',
                'Dockerfile example for a Python app',
                'Create a Docker image for Python application',
                'Best practices for writing Dockerfiles'
            ]
        };

        let trafficRunning = false;

        async function generateTraffic(type) {
            if (trafficRunning) return;
            trafficRunning = true;

            const count = parseInt(document.getElementById('trafficCount').value) || 10;
            const delay = parseInt(document.getElementById('trafficDelay').value) || 100;
            const status = document.getElementById('trafficStatus');
            const progress = document.getElementById('trafficProgress');
            const buttons = document.querySelectorAll('.traffic-presets button');

            buttons.forEach(b => b.disabled = true);
            const prompts = trafficPrompts[type];
            let hits = 0, misses = 0;

            for (let i = 0; i < count; i++) {
                const prompt = prompts[i % prompts.length];
                status.textContent = ` + "`" + `Sending ${i + 1}/${count}: "${prompt}"` + "`" + `;
                progress.style.width = ((i + 1) / count * 100) + '%';

                try {
                    const resp = await fetch('/v1/chat/completions', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({
                            model: document.getElementById('testModel').value,
                            messages: [{ role: 'user', content: prompt }]
                        })
                    });
                    const cacheStatus = resp.headers.get('X-Kallm-Cache');
                    if (cacheStatus === 'HIT') hits++; else misses++;
                    await resp.json();
                } catch (e) {
                    misses++;
                }

                if (delay > 0 && i < count - 1) {
                    await new Promise(r => setTimeout(r, delay));
                }
            }

            status.className = 'test-result';
            status.textContent = ` + "`" + `Complete! ${count} requests: ${hits} hits, ${misses} misses (${(hits/count*100).toFixed(1)}% hit rate)` + "`" + `;
            buttons.forEach(b => b.disabled = false);
            trafficRunning = false;
            fetchData();
        }

        // Allow Ctrl+Enter to send
        document.getElementById('testPrompt').addEventListener('keydown', (e) => {
            if (e.ctrlKey && e.key === 'Enter') sendTestPrompt();
        });

        // Logs functionality
        async function fetchLogs() {
            try {
                const resp = await fetch('/reports/logs');
                const logs = await resp.json();
                const container = document.getElementById('logsContainer');

                container.innerHTML = logs.map(log => {
                    const time = new Date(log.timestamp).toLocaleTimeString();
                    const cls = log.level === 'hit' ? 'hit' : log.level === 'miss' ? 'miss' : 'info';
                    return ` + "`" + `<div class="log-line ${cls}">[${time}] ${log.message}</div>` + "`" + `;
                }).join('');

                container.scrollTop = container.scrollHeight;
            } catch (e) {
                console.error('Failed to fetch logs:', e);
            }
        }

        async function clearLogs() {
            await fetch('/reports/logs/clear');
            document.getElementById('logsContainer').innerHTML = '';
        }

        fetchLogs();
        setInterval(fetchLogs, 2000);
    </script>
</body>
</html>`
}
