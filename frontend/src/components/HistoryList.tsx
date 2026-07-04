import { useState, useEffect, useCallback } from "react";
import { Search } from "lucide-react";
import {
  GetHistory,
  DeleteHistoryItem,
  ClearHistory,
  type HistoryEntry,
  type HistoryResponse,
} from "@/lib/api";
import { formatDate } from "@/lib/format";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const TASK_TYPE_LABELS: Record<string, string> = {
  search_choose: "Search & Download",
  add_lyrics: "Add Lyrics",
  link: "Link Download",
  download_link: "Link Download",
  embed_lrc: "Embed LRC",
  romanize_lrc: "Romanize LRC",
  extract_lrc: "Extract LRC",
  unknown: "Unknown",
};

export function HistoryList() {
  const [entries, setEntries] = useState<HistoryEntry[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [page, setPage] = useState(0);
  const limit = 30;

  const fetch = useCallback(async () => {
    setLoading(true);
    try {
      const res = (await GetHistory(
        limit,
        page * limit,
        statusFilter,
        search,
      )) as unknown as HistoryResponse;
      setEntries(res.entries ?? []);
      setTotal(res.total ?? 0);
    } catch {
      /* ignore */
    }
    setLoading(false);
  }, [limit, page, statusFilter, search]);

  useEffect(() => {
    fetch();
  }, [fetch]);

  const handleDelete = async (id: number) => {
    await DeleteHistoryItem(id);
    fetch();
  };

  const handleClear = async () => {
    if (!confirm("Clear all download history? This cannot be undone.")) return;
    await ClearHistory();
    setPage(0);
    fetch();
  };

  const totalPages = Math.max(1, Math.ceil(total / limit));

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-3">
        <div className="relative flex-1 min-w-[200px] max-w-sm">
          <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
          <input
            type="text"
            placeholder="Search history…"
            value={search}
            onChange={(e) => { setSearch(e.target.value); setPage(0); }}
            className="w-full pl-8 pr-3 py-2 rounded-lg bg-muted border border-border text-sm text-foreground
                       placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring/30"
          />
        </div>

        <Select
          value={statusFilter || "all"}
          onValueChange={(v) => { setStatusFilter(v === "all" ? "" : v); setPage(0); }}
        >
          <SelectTrigger className="w-40 bg-muted">
            <SelectValue placeholder="All status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All status</SelectItem>
            <SelectItem value="completed">Completed</SelectItem>
            <SelectItem value="failed">Failed</SelectItem>
          </SelectContent>
        </Select>

        <Button
          variant="ghost"
          onClick={handleClear}
          disabled={total === 0}
          className="text-destructive hover:text-destructive"
        >
          Clear all
        </Button>

        <span className="text-xs text-muted-foreground ml-auto">{total} entries</span>
      </div>

      {loading ? (
        <div className="flex justify-center py-12 text-muted-foreground text-sm">Loading…</div>
      ) : entries.length === 0 ? (
        <div className="flex flex-col items-center gap-3 py-12">
          <img src="/assets/image/lymuru-not-found.png" className="h-16 w-auto opacity-40" alt="" />
          <p className="text-muted-foreground text-sm">No history yet. Completed downloads will appear here.</p>
        </div>
      ) : (
        <>
          <div className="overflow-x-auto border border-border rounded-lg">
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-muted border-b border-border">
                  <th className="text-left px-4 py-2.5 text-muted-foreground font-medium">Task</th>
                  <th className="text-left px-4 py-2.5 text-muted-foreground font-medium hidden md:table-cell">Type</th>
                  <th className="text-left px-4 py-2.5 text-muted-foreground font-medium hidden sm:table-cell">Status</th>
                  <th className="text-left px-4 py-2.5 text-muted-foreground font-medium hidden lg:table-cell">Date</th>
                  <th className="text-right px-4 py-2.5 text-muted-foreground font-medium">Action</th>
                </tr>
              </thead>
              <tbody>
                {entries.map((entry) => (
                  <tr key={entry.id} className="border-b border-border last:border-b-0 hover:bg-muted/50 transition-colors">
                    <td className="px-4 py-2.5">
                      <span className="text-foreground truncate block max-w-[180px] sm:max-w-xs">
                        {entry.query || entry.task_id?.slice(0, 8)}
                      </span>
                      {entry.error && (
                        <span className="text-[10px] text-destructive block truncate max-w-[180px] sm:max-w-xs">
                          {entry.error}
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-2.5 text-muted-foreground hidden md:table-cell">
                      {TASK_TYPE_LABELS[entry.task_type] ?? entry.task_type}
                    </td>
                    <td className="px-4 py-2.5 hidden sm:table-cell">
                      <Badge variant={entry.status === "completed" ? "success" : "destructive"}>
                        {entry.status}
                      </Badge>
                    </td>
                    <td className="px-4 py-2.5 text-muted-foreground text-xs hidden lg:table-cell whitespace-nowrap">
                      {formatDate(entry.created_at)}
                    </td>
                    <td className="px-4 py-2.5 text-right">
                      <button
                        onClick={() => handleDelete(entry.id)}
                        className="text-xs text-destructive hover:opacity-80 transition-opacity"
                      >
                        Delete
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-2">
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setPage((p) => Math.max(0, p - 1))}
                disabled={page === 0}
              >
                Prev
              </Button>
              <span className="text-xs text-muted-foreground">
                {page + 1} / {totalPages}
              </span>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
                disabled={page >= totalPages - 1}
              >
                Next
              </Button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
