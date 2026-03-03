import { useState, type FormEvent, useEffect } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Dialog, DialogHeader, DialogTitle, DialogContent, DialogFooter } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { api } from "@/lib/api";
import type { ArticleLink } from "@/types/api";
import { Search } from "lucide-react";

interface BookOwlArticle {
  id: string;
  slug: string;
  title: string;
  excerpt: string;
}

interface LinkArticleModalProps {
  ticketId: number;
  open: boolean;
  onClose: () => void;
}

export function LinkArticleModal({ ticketId, open, onClose }: LinkArticleModalProps) {
  const [search, setSearch] = useState("");
  const [selectedId, setSelectedId] = useState("");
  const queryClient = useQueryClient();

  // Reset state when modal opens/closes.
  useEffect(() => {
    if (!open) {
      setSearch("");
      setSelectedId("");
    }
  }, [open]);

  const { data: results, isFetching } = useQuery({
    queryKey: ["search-articles", search],
    queryFn: () => api.get<BookOwlArticle[]>(`/search/articles?q=${encodeURIComponent(search)}`),
    enabled: open && search.length >= 2,
    placeholderData: (prev) => prev,
  });

  const mutation = useMutation({
    mutationFn: () =>
      api.post<ArticleLink>(`/tickets/${ticketId}/links/article`, {
        article_id: selectedId,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ticket-links", ticketId] });
      onClose();
    },
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!selectedId) return;
    mutation.mutate();
  }

  return (
    <Dialog open={open} onClose={onClose}>
      <DialogHeader>
        <DialogTitle>Link BookOwl Article</DialogTitle>
      </DialogHeader>
      <form onSubmit={handleSubmit}>
        <DialogContent className="space-y-3">
          <div className="relative">
            <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input
              value={search}
              onChange={(e) => {
                setSearch(e.target.value);
                setSelectedId("");
              }}
              placeholder="Search articles by title..."
              className="pl-9"
              autoFocus
            />
          </div>
          {search.length >= 2 && (
            <div className="max-h-48 overflow-y-auto rounded border border-border">
              {isFetching && !results?.length && (
                <p className="p-3 text-sm text-muted-foreground">Searching...</p>
              )}
              {results && results.length === 0 && !isFetching && (
                <p className="p-3 text-sm text-muted-foreground">No articles found</p>
              )}
              {results?.map((art) => (
                <button
                  key={art.id}
                  type="button"
                  onClick={() => setSelectedId(art.id)}
                  className={`w-full text-left px-3 py-2 text-sm hover:bg-accent/10 border-b border-border last:border-0 ${
                    selectedId === art.id ? "bg-accent/20" : ""
                  }`}
                >
                  <span className="block truncate font-medium">{art.title}</span>
                  {art.excerpt && (
                    <span className="block truncate text-xs text-muted-foreground">{art.excerpt}</span>
                  )}
                </button>
              ))}
            </div>
          )}
          {selectedId && (
            <p className="text-xs text-muted-foreground">
              Selected: <span className="font-mono">{selectedId.slice(0, 8)}...</span>
            </p>
          )}
          {mutation.isError && (
            <p className="text-sm text-destructive">
              {mutation.error instanceof Error ? mutation.error.message : "Failed to link article"}
            </p>
          )}
        </DialogContent>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" disabled={mutation.isPending || !selectedId}>
            {mutation.isPending ? "Linking..." : "Link Article"}
          </Button>
        </DialogFooter>
      </form>
    </Dialog>
  );
}
