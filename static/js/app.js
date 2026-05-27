// Poterie — Client-side JavaScript
// Vine animations, scroll observers, smooth interactions

document.addEventListener('DOMContentLoaded', () => {
  initVineAnimations();
  initHTMXAnimations();
});

// Scroll-driven vine growing effect
function initVineAnimations() {
  const svg = document.getElementById('vine-svg');
  if (!svg) return;

  const paths = svg.querySelectorAll('.vine-path');
  const leavesGroup = svg.querySelector('.vine-leaves');

  // Set initial dash offset
  paths.forEach(p => {
    const len = p.getTotalLength ? p.getTotalLength() : 2000;
    p.style.strokeDasharray = len;
    p.style.strokeDashoffset = len;
  });

  // Update vines based on scroll position
  const updateVines = () => {
    const scrollTop = window.scrollY;
    const maxScroll = Math.max(1, document.documentElement.scrollHeight - window.innerHeight);
    const progress = Math.min(1, scrollTop / maxScroll);

    paths.forEach(p => {
      const len = p.style.strokeDasharray;
      p.style.strokeDashoffset = len * (1 - progress * 1.5); // 1.5 for complete draw by 2/3 scroll
    });

    // Show leaves after 50% scroll
    if (leavesGroup) {
      leavesGroup.style.opacity = progress > 0.5 ? (progress - 0.5) * 2 : 0;
    }
  };

  // Initial call + scroll listener
  updateVines();

  let ticking = false;
  window.addEventListener('scroll', () => {
    if (!ticking) {
      requestAnimationFrame(() => {
        updateVines();
        ticking = false;
      });
      ticking = true;
    }
  }, { passive: true });
}

// HTMX event handlers for smooth transitions
function initHTMXAnimations() {
  document.body.addEventListener('htmx:afterSwap', (evt) => {
    // Animate newly swapped content
    const target = evt.detail.target;
    if (target) {
      target.classList.add('animate-fade-in');
      // Re-init scroll vine observer for new content
      setTimeout(initVineAnimations, 50);
    }
  });

  document.body.addEventListener('htmx:beforeRequest', () => {
    document.body.style.cursor = 'wait';
  });

  document.body.addEventListener('htmx:afterRequest', () => {
    document.body.style.cursor = '';
  });
}
