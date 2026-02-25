import { useEffect } from "react";

export function useTitle(page: string) {
  useEffect(() => {
    document.title = `${page} — TicketOwl`;
    return () => { document.title = "TicketOwl — Ticket Management Portal"; };
  }, [page]);
}
