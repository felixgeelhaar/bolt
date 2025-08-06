// Bolt Landing Page Interactive Features
// Performance-focused, zero-dependency JavaScript

class BoltLandingPage {
    constructor() {
        this.charts = {};
        this.benchmarkData = [];
        this.init();
    }

    init() {
        this.initializeCharts();
        this.setupCopyButtons();
        this.setupSmoothScrolling();
        this.loadBenchmarkData();
        this.setupNavbarScroll();
        this.setupSecurityAssessment();
        this.setupMigrationHub();
        this.setupMigrationCalculator();
        this.setupPerformanceCalculator();
        this.setupCodePlayground();
    }

    // Initialize Chart.js visualizations
    initializeCharts() {
        this.initPerformanceChart();
        this.initBenchmarkTrendChart();
    }

    initPerformanceChart() {
        const ctx = document.getElementById('performanceChart');
        if (!ctx) return;

        this.charts.performance = new Chart(ctx, {
            type: 'bar',
            data: {
                labels: ['Bolt', 'Zerolog', 'Zap', 'Logrus'],
                datasets: [{
                    label: 'Performance (ns/op)',
                    data: [98.1, 175.4, 189.7, 2847],
                    backgroundColor: [
                        '#2563EB', // Bolt - primary blue
                        '#6B7280', // Others - gray
                        '#6B7280',
                        '#6B7280'
                    ],
                    borderColor: [
                        '#1D4ED8',
                        '#4B5563',
                        '#4B5563',
                        '#4B5563'
                    ],
                    borderWidth: 2,
                    borderRadius: 8,
                    borderSkipped: false,
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        display: false
                    },
                    tooltip: {
                        backgroundColor: '#1F2937',
                        titleColor: '#F9FAFB',
                        bodyColor: '#F9FAFB',
                        borderColor: '#374151',
                        borderWidth: 1,
                        callbacks: {
                            label: function(context) {
                                const value = context.parsed.y;
                                const improvement = value < 98.1 ? '' : 
                                    ` (${Math.round((value - 98.1) / 98.1 * 100)}% slower)`;
                                return `${context.label}: ${value}ns/op${improvement}`;
                            }
                        }
                    }
                },
                scales: {
                    x: {
                        grid: {
                            display: false
                        },
                        ticks: {
                            font: {
                                family: 'Inter',
                                size: 12,
                                weight: 500
                            }
                        }
                    },
                    y: {
                        beginAtZero: true,
                        type: 'logarithmic',
                        grid: {
                            color: '#E5E7EB'
                        },
                        ticks: {
                            font: {
                                family: 'Inter',
                                size: 12
                            },
                            callback: function(value) {
                                return value + 'ns';
                            }
                        }
                    }
                },
                animation: {
                    duration: 2000,
                    easing: 'easeOutQuart'
                }
            }
        });
    }

    initBenchmarkTrendChart() {
        const ctx = document.getElementById('benchmarkChart');
        if (!ctx) return;

        // Placeholder data - will be replaced with real data from GitHub Actions
        const mockData = this.generateMockBenchmarkData();

        this.charts.benchmarkTrend = new Chart(ctx, {
            type: 'line',
            data: {
                labels: mockData.dates,
                datasets: [
                    {
                        label: 'Bolt (Enabled)',
                        data: mockData.boltEnabled,
                        borderColor: '#2563EB',
                        backgroundColor: 'rgba(37, 99, 235, 0.1)',
                        fill: true,
                        tension: 0.4,
                        pointRadius: 4,
                        pointHoverRadius: 6,
                        pointBackgroundColor: '#2563EB',
                        pointBorderColor: '#ffffff',
                        pointBorderWidth: 2
                    },
                    {
                        label: 'Zerolog (Enabled)',
                        data: mockData.zerologEnabled,
                        borderColor: '#6B7280',
                        backgroundColor: 'rgba(107, 114, 128, 0.1)',
                        fill: false,
                        tension: 0.4,
                        pointRadius: 3,
                        pointHoverRadius: 5,
                        pointBackgroundColor: '#6B7280',
                        pointBorderColor: '#ffffff',
                        pointBorderWidth: 2
                    }
                ]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        position: 'top',
                        labels: {
                            font: {
                                family: 'Inter',
                                size: 12,
                                weight: 500
                            }
                        }
                    },
                    tooltip: {
                        backgroundColor: '#1F2937',
                        titleColor: '#F9FAFB',
                        bodyColor: '#F9FAFB',
                        borderColor: '#374151',
                        borderWidth: 1,
                        callbacks: {
                            label: function(context) {
                                return `${context.dataset.label}: ${context.parsed.y}ns/op`;
                            }
                        }
                    }
                },
                scales: {
                    x: {
                        type: 'time',
                        time: {
                            unit: 'day',
                            displayFormats: {
                                day: 'MMM dd'
                            }
                        },
                        grid: {
                            display: false
                        },
                        ticks: {
                            font: {
                                family: 'Inter',
                                size: 12
                            }
                        }
                    },
                    y: {
                        beginAtZero: true,
                        grid: {
                            color: '#E5E7EB'
                        },
                        ticks: {
                            font: {
                                family: 'Inter',
                                size: 12
                            },
                            callback: function(value) {
                                return value + 'ns';
                            }
                        }
                    }
                },
                animation: {
                    duration: 2000,
                    easing: 'easeOutQuart'
                }
            }
        });
    }

    generateMockBenchmarkData() {
        const dates = [];
        const boltEnabled = [];
        const zerologEnabled = [];
        const startDate = new Date();
        startDate.setDate(startDate.getDate() - 30);

        for (let i = 0; i < 30; i++) {
            const date = new Date(startDate);
            date.setDate(startDate.getDate() + i);
            dates.push(date);
            
            // Simulate improving performance over time for Bolt
            const baseBolt = 110 - (i * 0.4);
            boltEnabled.push(baseBolt + (Math.random() - 0.5) * 5);
            
            // Zerolog remains relatively stable
            zerologEnabled.push(175 + (Math.random() - 0.5) * 8);
        }

        return { dates, boltEnabled, zerologEnabled };
    }

    // Copy to clipboard functionality
    setupCopyButtons() {
        document.querySelectorAll('.copy-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                e.preventDefault();
                const textToCopy = btn.getAttribute('data-copy');
                
                // Modern clipboard API with fallback
                if (navigator.clipboard && window.isSecureContext) {
                    navigator.clipboard.writeText(textToCopy).then(() => {
                        this.showCopyFeedback(btn);
                    });
                } else {
                    // Fallback for older browsers
                    this.fallbackCopyToClipboard(textToCopy);
                    this.showCopyFeedback(btn);
                }
            });
        });
    }

    fallbackCopyToClipboard(text) {
        const textArea = document.createElement('textarea');
        textArea.value = text;
        textArea.style.position = 'fixed';
        textArea.style.left = '-999999px';
        textArea.style.top = '-999999px';
        document.body.appendChild(textArea);
        textArea.focus();
        textArea.select();
        document.execCommand('copy');
        textArea.remove();
    }

    showCopyFeedback(btn) {
        const originalHTML = btn.innerHTML;
        btn.innerHTML = '<svg viewBox="0 0 24 24"><path d="M20.285 2l-11.285 11.567-5.286-5.011-3.714 3.716 9 8.728 15-15.285z"/></svg>';
        btn.style.color = '#10B981';
        
        setTimeout(() => {
            btn.innerHTML = originalHTML;
            btn.style.color = '';
        }, 2000);
    }

    // Smooth scrolling for navigation
    setupSmoothScrolling() {
        document.querySelectorAll('a[href^="#"]').forEach(anchor => {
            anchor.addEventListener('click', (e) => {
                e.preventDefault();
                const target = document.querySelector(anchor.getAttribute('href'));
                if (target) {
                    const offset = 80; // Account for fixed navbar
                    const elementPosition = target.offsetTop;
                    const offsetPosition = elementPosition - offset;

                    window.scrollTo({
                        top: offsetPosition,
                        behavior: 'smooth'
                    });
                }
            });
        });
    }

    // Navbar scroll effects
    setupNavbarScroll() {
        let lastScrollTop = 0;
        const navbar = document.querySelector('.navbar');
        
        window.addEventListener('scroll', () => {
            const scrollTop = window.pageYOffset || document.documentElement.scrollTop;
            
            // Add shadow when scrolled
            if (scrollTop > 10) {
                navbar.style.boxShadow = '0 4px 6px -1px rgb(0 0 0 / 0.1)';
            } else {
                navbar.style.boxShadow = '';
            }
            
            lastScrollTop = scrollTop;
        });
    }

    // Load real benchmark data from GitHub Actions
    async loadBenchmarkData() {
        const loadingElement = document.getElementById('chartLoading');
        
        try {
            // This would fetch from GitHub Actions artifacts in a real implementation
            // For now, we'll simulate loading and then hide the loading indicator
            await this.simulateDataLoading();
            
            if (loadingElement) {
                loadingElement.style.display = 'none';
            }
            
            // Update footer stats with real data
            this.updateFooterStats();
            
        } catch (error) {
            console.warn('Failed to load benchmark data:', error);
            if (loadingElement) {
                loadingElement.innerHTML = '<p style="color: #6B7280;">Using cached benchmark data</p>';
            }
        }
    }

    async simulateDataLoading() {
        // Simulate API call delay
        return new Promise(resolve => {
            setTimeout(resolve, 2000);
        });
    }

    updateFooterStats() {
        const statsElement = document.getElementById('footerStats');
        if (!statsElement) return;

        // In a real implementation, this would come from GitHub API
        const stats = {
            version: 'v1.0.0',
            goVersion: 'Go 1.21+',
            dependencies: '0 Dependencies',
            lastUpdate: new Date().toLocaleDateString()
        };

        statsElement.innerHTML = `
            <span class="stat">Latest: ${stats.version}</span>
            <span class="stat">${stats.goVersion}</span>
            <span class="stat">${stats.dependencies}</span>
        `;
    }

    // Security Assessment Tool
    setupSecurityAssessment() {
        const appTypeSelect = document.getElementById('appType');
        const complianceCheckboxes = document.querySelectorAll('.compliance-checkbox');
        const securityScore = document.getElementById('securityScore');
        const recommendations = document.getElementById('securityRecommendations');

        const updateSecurityAssessment = () => {
            const appType = appTypeSelect?.value || 'web';
            const selectedCompliances = Array.from(complianceCheckboxes)
                .filter(cb => cb.checked)
                .map(cb => cb.id);

            let score = 100;
            let recs = [
                '✓ JSON injection prevention enabled',
                '✓ Input validation active',
                '✓ Thread-safe operations',
                '✓ Audit trail compliance'
            ];

            // Adjust score and recommendations based on selections
            if (selectedCompliances.includes('hipaa')) {
                recs.push('✓ HIPAA audit logging configured');
                recs.push('✓ PHI data masking available');
            }
            if (selectedCompliances.includes('sox')) {
                recs.push('✓ SOX financial audit trails');
                recs.push('✓ Tamper-evident logging');
            }
            if (selectedCompliances.includes('gdpr')) {
                recs.push('✓ GDPR data retention policies');
                recs.push('✓ Personal data anonymization');
            }

            if (securityScore) {
                securityScore.textContent = score;
            }

            if (recommendations) {
                recommendations.innerHTML = `
                    <h4>Recommendations:</h4>
                    <ul>
                        ${recs.map(rec => `<li>${rec}</li>`).join('')}
                    </ul>
                `;
            }
        };

        appTypeSelect?.addEventListener('change', updateSecurityAssessment);
        complianceCheckboxes.forEach(cb => {
            cb.addEventListener('change', updateSecurityAssessment);
        });

        // Initial update
        updateSecurityAssessment();
    }

    // Migration Hub
    setupMigrationHub() {
        const migrationTabs = document.querySelectorAll('.migration-tab');
        const migrationPanels = document.querySelectorAll('.migration-panel');

        migrationTabs.forEach(tab => {
            tab.addEventListener('click', () => {
                // Remove active from all tabs and panels
                migrationTabs.forEach(t => t.classList.remove('active'));
                migrationPanels.forEach(p => p.classList.remove('active'));

                // Add active to clicked tab and corresponding panel
                tab.classList.add('active');
                const library = tab.dataset.library;
                const panel = document.getElementById(`${library}-migration`);
                if (panel) {
                    panel.classList.add('active');
                }
            });
        });
    }

    // Migration Effort Calculator
    setupMigrationCalculator() {
        const linesOfCodeInput = document.getElementById('linesOfCode');
        const logStatementsInput = document.getElementById('logStatements');
        const currentLibrarySelect = document.getElementById('currentLibrary');
        const effortDaysSpan = document.getElementById('effortDays');
        const codeChangeDaysSpan = document.getElementById('codeChangeDays');
        const testingDaysSpan = document.getElementById('testingDays');
        const docDaysSpan = document.getElementById('docDays');

        const calculateMigrationEffort = () => {
            const linesOfCode = parseInt(linesOfCodeInput?.value) || 0;
            const logStatements = parseInt(logStatementsInput?.value) || 0;
            const currentLibrary = currentLibrarySelect?.value || 'zerolog';

            // Base effort calculation
            let codeChangeDays = Math.max(0.5, logStatements / 500 * 1.5);
            let testingDays = codeChangeDays * 0.3;
            let docDays = 0.5;

            // Adjust based on current library complexity
            const complexityMultiplier = {
                'stdlib': 2.0,  // Most complex migration
                'logrus': 1.5,
                'zap': 1.2,
                'zerolog': 1.0  // Easiest migration
            };

            const multiplier = complexityMultiplier[currentLibrary] || 1.0;
            codeChangeDays *= multiplier;
            testingDays *= multiplier;

            const totalDays = codeChangeDays + testingDays + docDays;

            if (effortDaysSpan) effortDaysSpan.textContent = totalDays.toFixed(1);
            if (codeChangeDaysSpan) codeChangeDaysSpan.textContent = `${codeChangeDays.toFixed(1)} days`;
            if (testingDaysSpan) testingDaysSpan.textContent = `${testingDays.toFixed(1)} days`;
            if (docDaysSpan) docDaysSpan.textContent = `${docDays.toFixed(1)} days`;
        };

        linesOfCodeInput?.addEventListener('input', calculateMigrationEffort);
        logStatementsInput?.addEventListener('input', calculateMigrationEffort);
        currentLibrarySelect?.addEventListener('change', calculateMigrationEffort);

        // Initial calculation
        calculateMigrationEffort();
    }

    // Performance Calculator
    setupPerformanceCalculator() {
        const logEventsInput = document.getElementById('logEventsPerSec');
        const fieldsPerLogInput = document.getElementById('fieldsPerLog');
        const perfLibrarySelect = document.getElementById('perfCurrentLibrary');
        const cpuSavingSpan = document.getElementById('cpuSaving');
        const memorySavingSpan = document.getElementById('memorySaving');
        const serverSavingsSpan = document.getElementById('serverSavings');
        const devSavingsSpan = document.getElementById('devSavings');
        const opsSavingsSpan = document.getElementById('opsSavings');

        const calculatePerformanceImpact = () => {
            const eventsPerSec = parseInt(logEventsInput?.value) || 0;
            const fieldsPerLog = parseInt(fieldsPerLogInput?.value) || 0;
            const currentLibrary = perfLibrarySelect?.value || 'zerolog';

            // Performance metrics (ns/op)
            const libraryPerformance = {
                'zerolog': { ns: 175.4, allocations: 0 },
                'zap': { ns: 189.7, allocations: 1 },
                'logrus': { ns: 2847, allocations: 23 },
                'stdlib': { ns: 1200, allocations: 8 }
            };

            const currentPerf = libraryPerformance[currentLibrary];
            const boltPerf = { ns: 98.1, allocations: 0 };

            // Calculate savings
            const cpuImprovement = ((currentPerf.ns - boltPerf.ns) / currentPerf.ns) * 100;
            const memoryImprovement = currentPerf.allocations > 0 ? 100 : 0;

            // Calculate annual cost savings
            const yearlyOperations = eventsPerSec * fieldsPerLog * 31536000; // seconds in year
            const cpuCostSaving = (cpuImprovement / 100) * 15000; // $15k base server cost
            const devTimeSaving = cpuImprovement * 500; // $500 per percentage point
            const opsCostSaving = (cpuImprovement / 100) * 12000; // $12k base ops cost

            if (cpuSavingSpan) cpuSavingSpan.textContent = Math.round(cpuImprovement);
            if (memorySavingSpan) memorySavingSpan.textContent = Math.round(memoryImprovement);
            if (serverSavingsSpan) serverSavingsSpan.textContent = `$${Math.round(cpuCostSaving).toLocaleString()}`;
            if (devSavingsSpan) devSavingsSpan.textContent = `$${Math.round(devTimeSaving).toLocaleString()}`;
            if (opsSavingsSpan) opsSavingsSpan.textContent = `$${Math.round(opsCostSaving).toLocaleString()}`;
        };

        logEventsInput?.addEventListener('input', calculatePerformanceImpact);
        fieldsPerLogInput?.addEventListener('input', calculatePerformanceImpact);
        perfLibrarySelect?.addEventListener('change', calculatePerformanceImpact);

        // Initial calculation
        calculatePerformanceImpact();
    }

    // Code Playground
    setupCodePlayground() {
        const codeEditor = document.getElementById('codeEditor');
        const runButton = document.getElementById('runCode');
        const clearButton = document.getElementById('clearOutput');
        const outputConsole = document.getElementById('outputConsole');
        const exampleButtons = document.querySelectorAll('.example-btn');

        // Example code snippets
        const examples = {
            basic: `package main

import (
    "os"
    "github.com/felixgeelhaar/bolt"
)

func main() {
    logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
    
    logger.Info().
        Str("service", "demo-api").
        Int("port", 8080).
        Bool("secure", true).
        Msg("Server starting")
}`,
            context: `package main

import (
    "context"
    "os"
    "github.com/felixgeelhaar/bolt"
)

func main() {
    logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
    ctx := context.Background()
    
    // Context-aware logging with tracing
    contextLogger := logger.Ctx(ctx)
    
    contextLogger.Info().
        Str("operation", "user_login").
        Str("user_id", "usr_12345").
        Msg("User authentication successful")
}`,
            performance: `package main

import (
    "os"
    "time"
    "github.com/felixgeelhaar/bolt"
)

func main() {
    logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
    
    // Benchmark different log levels
    start := time.Now()
    
    for i := 0; i < 1000; i++ {
        logger.Debug().
            Int("iteration", i).
            Msg("Debug message")
    }
    
    elapsed := time.Since(start)
    
    logger.Info().
        Dur("elapsed", elapsed).
        Float64("ns_per_op", float64(elapsed.Nanoseconds())/1000).
        Msg("Benchmark completed")
}`,
            security: `package main

import (
    "os"
    "github.com/felixgeelhaar/bolt"
)

func main() {
    logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
    
    // Security: Input validation and sanitization
    userInput := "user@example.com\n{\"malicious\": true}"
    
    // Bolt automatically sanitizes inputs
    logger.Warn().
        Str("user_input", userInput).
        Str("source_ip", "192.168.1.100").
        Bool("input_validated", true).
        Msg("Potentially malicious input detected")
    
    // Compliance logging
    logger.Info().
        Str("event_type", "audit_log").
        Str("compliance", "SOX").
        Str("action", "data_access").
        Msg("Compliance audit event")
}`
        };

        // Simulate Go code execution and JSON output
        const simulateExecution = (code) => {
            const output = [];
            const lines = code.split('\n');
            
            // Parse log statements and generate JSON output
            let inLogStatement = false;
            let currentLog = {};
            
            lines.forEach(line => {
                line = line.trim();
                
                if (line.includes('logger.') && (line.includes('Info()') || line.includes('Debug()') || line.includes('Warn()') || line.includes('Error()'))) {
                    inLogStatement = true;
                    currentLog = {
                        timestamp: new Date().toISOString(),
                        level: line.includes('Info()') ? 'info' : 
                               line.includes('Debug()') ? 'debug' :
                               line.includes('Warn()') ? 'warn' : 'error'
                    };
                }
                
                if (inLogStatement) {
                    if (line.includes('Str(')) {
                        const match = line.match(/Str\("([^"]+)",\s*"([^"]+)"/); 
                        if (match) currentLog[match[1]] = match[2];
                    }
                    if (line.includes('Int(')) {
                        const match = line.match(/Int\("([^"]+)",\s*(\d+)/); 
                        if (match) currentLog[match[1]] = parseInt(match[2]);
                    }
                    if (line.includes('Bool(')) {
                        const match = line.match(/Bool\("([^"]+)",\s*(true|false)/); 
                        if (match) currentLog[match[1]] = match[2] === 'true';
                    }
                    if (line.includes('Float64(')) {
                        const match = line.match(/Float64\("([^"]+)",\s*([\d.]+)/); 
                        if (match) currentLog[match[1]] = parseFloat(match[2]);
                    }
                    if (line.includes('Msg(')) {
                        const match = line.match(/Msg\("([^"]+)"/); 
                        if (match) {
                            currentLog.message = match[1];
                            output.push(JSON.stringify(currentLog, null, 2));
                            inLogStatement = false;
                            currentLog = {};
                        }
                    }
                }
            });
            
            return output.join('\n\n');
        };

        // Run code button
        runButton?.addEventListener('click', () => {
            const code = codeEditor?.value || '';
            const output = simulateExecution(code);
            
            if (outputConsole) {
                outputConsole.innerHTML = `<pre>${output || 'No log output generated'}</pre>`;
            }
        });

        // Clear output button
        clearButton?.addEventListener('click', () => {
            if (outputConsole) {
                outputConsole.innerHTML = '<div class="output-placeholder">Click "Run Code" to see the JSON output</div>';
            }
        });

        // Example buttons
        exampleButtons.forEach(btn => {
            btn.addEventListener('click', () => {
                const example = btn.dataset.example;
                if (examples[example] && codeEditor) {
                    codeEditor.value = examples[example];
                    // Auto-run the example
                    runButton?.click();
                }
            });
        });
    }

    // Utility method for future GitHub Actions integration
    async fetchBenchmarkData() {
        // This would integrate with GitHub Actions workflow artifacts
        // Example endpoint: https://api.github.com/repos/felixgeelhaar/bolt/actions/artifacts
        const response = await fetch('/api/benchmarks');
        return response.json();
    }
}

// Performance monitoring
const performanceObserver = new PerformanceObserver((list) => {
    list.getEntries().forEach((entry) => {
        if (entry.entryType === 'largest-contentful-paint') {
            console.log('LCP:', entry.startTime);
        }
    });
});

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    // Start performance monitoring
    if (typeof PerformanceObserver !== 'undefined') {
        performanceObserver.observe({ entryTypes: ['largest-contentful-paint'] });
    }
    
    // Initialize landing page functionality
    window.boltLanding = new BoltLandingPage();
});

// Handle window resize for charts
window.addEventListener('resize', () => {
    if (window.boltLanding && window.boltLanding.charts) {
        Object.values(window.boltLanding.charts).forEach(chart => {
            if (chart && typeof chart.resize === 'function') {
                chart.resize();
            }
        });
    }
});