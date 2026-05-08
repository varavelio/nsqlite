// Open llms.txt and llms-full.txt links in a new tab
document.addEventListener("DOMContentLoaded", () => {
  const llms = document.querySelectorAll('a[href$="llms.txt"]');
  const llmsFull = document.querySelectorAll('a[href$="llms-full.txt"]');
  [...llms, ...llmsFull].forEach((link) => {
    link.setAttribute("target", "_blank");
    link.setAttribute("rel", "noopener");
  });
});
