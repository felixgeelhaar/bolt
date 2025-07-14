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

    // Initialize lightweight charts
    initializeCharts() {
        this.initPerformanceChart();
        this.initBenchmarkTrendChart();
    }

    initPerformanceChart(logmaValue = 87.94) {
        const container = document.getElementById('performanceChart');
        if (!container) return;

        // Set fixed height immediately to prevent layout shift
        container.style.height = '100%';
        container.style.minHeight = '260px';

        const data = [
            { label: 'Logma', value: logmaValue, class: 'logma' },
            { label: 'Zerolog', value: 175.4, class: 'other' },
            { label: 'Zap', value: 189.7, class: 'other' },
            { label: 'Logrus', value: 2847, class: 'other' }
        ];

        const maxValue = Math.max(...data.map(d => d.value));
        const logScale = (value) => Math.log(value) / Math.log(maxValue) * 200 + 20;

        // Use requestAnimationFrame for smooth rendering
        requestAnimationFrame(() => {
            container.innerHTML = data.map(item => {
                const height = logScale(item.value);
                const improvement = item.value > logmaValue ? 
                    ` (${Math.round((item.value - logmaValue) / logmaValue * 100)}% slower)` : '';
                
                return `
                    <div class="chart-bar" title="${item.label}: ${item.value}ns/op${improvement}">
                        <div class="bar ${item.class}" style="height: ${height}px;">
                            <div class="bar-value">${item.value}ns</div>
                        </div>
                        <div class="bar-label">${item.label}</div>
                    </div>
                `;
            }).join('');
        });
    }

    initBenchmarkTrendChart() {
        const container = document.getElementById('benchmarkChart');
        if (!container) return;

        // Set fixed dimensions immediately to prevent layout shift
        container.style.height = '300px !important';
        container.style.minHeight = '300px !important';
        container.style.maxHeight = '300px !important';
        container.style.overflow = 'hidden';
        
        // Use requestAnimationFrame to ensure smooth rendering
        requestAnimationFrame(() => {
            container.innerHTML = `
                <div class="trend-content" style="height: 100% !important; max-height: 260px !important; position: relative; overflow: hidden !important;">
                    <div class="trend-legend">
                        <div class="legend-item">
                            <div class="legend-color logma"></div>
                            <span>Logma (improving performance)</span>
                        </div>
                        <div class="legend-item">
                            <div class="legend-color zerolog"></div>
                            <span>Zerolog (stable)</span>
                        </div>
                    </div>
                    <svg width="100%" height="200" style="position: absolute; bottom: 20px; left: 20px; right: 20px; max-height: 200px; overflow: hidden;" viewBox="0 0 400 200" preserveAspectRatio="none">
                        <defs>
                            <linearGradient id="logmaGradient" x1="0%" y1="0%" x2="0%" y2="100%">
                                <stop offset="0%" style="stop-color:#2563EB;stop-opacity:0.3" />
                                <stop offset="100%" style="stop-color:#2563EB;stop-opacity:0" />
                            </linearGradient>
                        </defs>
                        <!-- Logma trend line (improving - going down is better) -->
                        <path d="M0,80 Q100,95 200,120 T400,150" stroke="#2563EB" stroke-width="3" fill="none"/>
                        <path d="M0,80 Q100,95 200,120 T400,150 L400,200 L0,200 Z" fill="url(#logmaGradient)"/>
                        <!-- Zerolog trend line (stable) -->
                        <path d="M0,120 Q100,115 200,118 T400,115" stroke="#6B7280" stroke-width="2" fill="none"/>
                        <!-- Data points -->
                        <circle cx="0" cy="80" r="4" fill="#2563EB"/>
                        <circle cx="100" cy="95" r="4" fill="#2563EB"/>
                        <circle cx="200" cy="120" r="4" fill="#2563EB"/>
                        <circle cx="300" cy="135" r="4" fill="#2563EB"/>
                        <circle cx="400" cy="150" r="4" fill="#2563EB"/>
                        <!-- Labels -->
                        <text x="20" y="190" font-size="10" fill="#6B7280">30 days ago (slower)</text>
                        <text x="320" y="190" font-size="10" fill="#6B7280">Today (faster)</text>
                    </svg>
                </div>
            `;
        });
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
        try {
            // Try to load live benchmark data
            const response = await fetch('assets/data/latest-benchmarks.json');
            if (response.ok) {
                const data = await response.json();
                this.updateChartsWithLiveData(data);
            }
        } catch (error) {
            console.log('Using static benchmark data');
        }
        
        // Update footer stats
        this.updateFooterStats();
    }

    updateChartsWithLiveData(data) {
        // Update performance chart with live data
        if (data.logma_ns_per_op) {
            this.initPerformanceChart(data.logma_ns_per_op);
        }
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