import { SearchTab } from "@/components/SearchTab";
import { LinkDownloadTab } from "@/components/LinkDownloadTab";
import { BulkDownloadTab } from "@/components/BulkDownloadTab";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Link2, ListMusic, Search as SearchIcon } from "lucide-react";

export function HomePage() {
  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <div className="flex items-center gap-3">
            <img src="/assets/image/lymuru-mascot.png" alt="" className="h-10 w-10" />
            <div>
              <CardTitle>Welcome to Lymuru</CardTitle>
              <CardDescription>
                Search Spotify/Deezer, paste a link, or submit many links at once.
              </CardDescription>
            </div>
          </div>
        </CardHeader>
      </Card>

      <Tabs defaultValue="search" className="w-full">
        <TabsList className="w-full sm:w-auto">
          <TabsTrigger value="search" className="gap-1.5">
            <SearchIcon className="h-3.5 w-3.5" />
            Search
          </TabsTrigger>
          <TabsTrigger value="link" className="gap-1.5">
            <Link2 className="h-3.5 w-3.5" />
            Link
          </TabsTrigger>
          <TabsTrigger value="bulk" className="gap-1.5">
            <ListMusic className="h-3.5 w-3.5" />
            Bulk
          </TabsTrigger>
        </TabsList>
        <TabsContent value="search" className="mt-4">
          <SearchTab />
        </TabsContent>
        <TabsContent value="link" className="mt-4">
          <LinkDownloadTab />
        </TabsContent>
        <TabsContent value="bulk" className="mt-4">
          <BulkDownloadTab />
        </TabsContent>
      </Tabs>
    </div>
  );
}
