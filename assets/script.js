// Bolt landing page — minimal vanilla JS.
// Two responsibilities only:
//   1. Theme toggle (system → user override → localStorage persistence)
//   2. Copy-to-clipboard buttons
//
// Anything else (Chart.js, fake "trend data", PerformanceObserver,
// simulated loaders) was removed in the 2026 redesign. The previous
// script set up a perf chart against fabricated values which leaked
// trust as soon as anyone read the source. The bench table in
// index.html is now the source of truth.

(() => {
  'use strict';

  // ---- Theme ---------------------------------------------------------------
  const THEME_KEY = 'bolt-theme';
  const root = document.documentElement;

  function applyStoredTheme() {
    try {
      const stored = localStorage.getItem(THEME_KEY);
      if (stored === 'dark' || stored === 'light') {
        root.setAttribute('data-theme', stored);
      }
    } catch (_) {
      // localStorage may be blocked (private mode); no-op.
    }
  }

  function currentResolvedTheme() {
    const explicit = root.getAttribute('data-theme');
    if (explicit) return explicit;
    return window.matchMedia('(prefers-color-scheme: dark)').matches
      ? 'dark'
      : 'light';
  }

  function toggleTheme() {
    const next = currentResolvedTheme() === 'dark' ? 'light' : 'dark';
    root.setAttribute('data-theme', next);
    try {
      localStorage.setItem(THEME_KEY, next);
    } catch (_) {
      // ignore
    }
  }

  applyStoredTheme();
  document.addEventListener('DOMContentLoaded', () => {
    const btn = document.querySelector('.theme-toggle');
    if (btn) {
      btn.addEventListener('click', toggleTheme);
    }

    // ---- Copy buttons -----------------------------------------------------
    document.querySelectorAll('.copy-btn[data-copy]').forEach((btn) => {
      btn.addEventListener('click', async () => {
        const id = btn.getAttribute('data-copy');
        const node = document.getElementById(id);
        if (!node) return;
        const text = node.innerText.trim();
        try {
          await navigator.clipboard.writeText(text);
          const orig = btn.textContent;
          btn.textContent = 'Copied';
          btn.classList.add('copied');
          setTimeout(() => {
            btn.textContent = orig;
            btn.classList.remove('copied');
          }, 1400);
        } catch (_) {
          // Clipboard API blocked. Fall back to selection.
          const range = document.createRange();
          range.selectNodeContents(node);
          const sel = window.getSelection();
          sel.removeAllRanges();
          sel.addRange(range);
        }
      });
    });
  });
})();
