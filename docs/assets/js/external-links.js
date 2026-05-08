// Open specified links in a new tab
document.addEventListener("DOMContentLoaded", () => {
  const links = [
    "llms.txt",
    "llms-full.txt",
    "schema/explorer.html",
    "schema/swagger.html",
    "schema/openapi.yaml",
  ];

  const selector = links.map((link) => `a[href$="${link}"]`).join(",");

  document.querySelectorAll(selector).forEach((link) => {
    link.setAttribute("target", "_blank");
    link.setAttribute("rel", "noopener");
  });
});
