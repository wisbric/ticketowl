import { useState, type FormEvent } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Dialog, DialogHeader, DialogTitle, DialogContent, DialogFooter } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { api } from "@/lib/api";
import type { ArticleLink } from "@/types/api";

interface LinkArticleModalProps {
  ticketId: number;
  open: boolean;
  onClose: () => void;
}

export function LinkArticleModal({ ticketId, open, onClose }: LinkArticleModalProps) {
  const [articleId, setArticleId] = useState("");
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: () =>
      api.post<ArticleLink>(`/tickets/${ticketId}/links/article`, {
        article_id: articleId,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ticket-links", ticketId] });
      setArticleId("");
      onClose();
    },
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!articleId.trim()) return;
    mutation.mutate();
  }

  return (
    <Dialog open={open} onClose={onClose}>
      <DialogHeader>
        <DialogTitle>Link BookOwl Article</DialogTitle>
      </DialogHeader>
      <form onSubmit={handleSubmit}>
        <DialogContent>
          <label htmlFor="article-id" className="mb-1 block text-sm font-medium">
            Article ID
          </label>
          <Input
            id="article-id"
            value={articleId}
            onChange={(e) => setArticleId(e.target.value)}
            placeholder="Enter BookOwl article UUID"
            required
          />
          {mutation.isError && (
            <p className="mt-2 text-sm text-destructive">
              {mutation.error instanceof Error ? mutation.error.message : "Failed to link article"}
            </p>
          )}
        </DialogContent>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" disabled={mutation.isPending}>
            {mutation.isPending ? "Linking..." : "Link Article"}
          </Button>
        </DialogFooter>
      </form>
    </Dialog>
  );
}
