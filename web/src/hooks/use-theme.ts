import { useCallback, useSyncExternalStore } from "react";

function getSnapshot(): "dark" | "light" {
  return document.documentElement.classList.contains("light") ? "light" : "dark";
}

function subscribe(cb: () => void) {
  const observer = new MutationObserver(cb);
  observer.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] });
  return () => observer.disconnect();
}

export function useTheme() {
  const theme = useSyncExternalStore(subscribe, getSnapshot, () => "dark" as const);

  const toggle = useCallback(() => {
    const next = theme === "dark" ? "light" : "dark";
    document.documentElement.classList.toggle("dark", next === "dark");
    document.documentElement.classList.toggle("light", next === "light");
    localStorage.setItem("ticketowl_theme", next);
  }, [theme]);

  return { theme, toggle };
}

export function initTheme() {
  const stored = localStorage.getItem("ticketowl_theme");
  const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
  const isDark = stored ? stored === "dark" : prefersDark;
  document.documentElement.classList.toggle("dark", isDark);
  document.documentElement.classList.toggle("light", !isDark);
}
