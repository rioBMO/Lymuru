import { HistoryList } from "@/components/HistoryList";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";

export function HistoryPage() {
  return (
    <div className="space-y-4 max-w-4xl">
      <Card>
        <CardHeader>
          <CardTitle>Download History</CardTitle>
          <CardDescription>
            Completed and failed downloads, most recent first.
          </CardDescription>
        </CardHeader>
      </Card>
      <Card>
        <CardContent className="p-5">
          <HistoryList />
        </CardContent>
      </Card>
    </div>
  );
}
