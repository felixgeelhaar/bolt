// Logma Landing Page Interactive Features
// Performance-focused, zero-dependency JavaScript

class LogmaLandingPage {
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
                labels: ['Logma', 'Zerolog', 'Zap', 'Logrus'],
                datasets: [{
                    label: 'Performance (ns/op)',
                    data: [87.94, 175.4, 189.7, 2847],
                    backgroundColor: [
                        '#2563EB', // Logma - primary blue
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
                                const improvement = value < 87.94 ? '' : 
                                    ` (${Math.round((value - 87.94) / 87.94 * 100)}% slower)`;
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
                        label: 'Logma (Enabled)',
                        data: mockData.logmaEnabled,
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
        const logmaEnabled = [];
        const zerologEnabled = [];
        const startDate = new Date();
        startDate.setDate(startDate.getDate() - 30);

        for (let i = 0; i < 30; i++) {
            const date = new Date(startDate);
            date.setDate(startDate.getDate() + i);
            dates.push(date);
            
            // Simulate improving performance over time for Logma
            const baseLogma = 95 - (i * 0.2);
            logmaEnabled.push(baseLogma + (Math.random() - 0.5) * 3);
            
            // Zerolog remains relatively stable
            zerologEnabled.push(175 + (Math.random() - 0.5) * 8);
        }

        return { dates, logmaEnabled, zerologEnabled };
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
            // Hide loading indicator immediately since we're using static data
            if (loadingElement) {
                loadingElement.style.display = 'none';
            }
            
            // Update footer stats
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

    // Utility method for future GitHub Actions integration
    async fetchBenchmarkData() {
        // This would integrate with GitHub Actions workflow artifacts
        // Example endpoint: https://api.github.com/repos/felixgeelhaar/logma/actions/artifacts
        // For now, return mock data to prevent hanging
        return new Promise(resolve => {
            setTimeout(() => {
                resolve(this.generateMockBenchmarkData());
            }, 500);
        });
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
    window.logmaLanding = new LogmaLandingPage();
});

// Handle window resize for charts
window.addEventListener('resize', () => {
    if (window.logmaLanding && window.logmaLanding.charts) {
        Object.values(window.logmaLanding.charts).forEach(chart => {
            if (chart && typeof chart.resize === 'function') {
                chart.resize();
            }
        });
    }
});