import { AddLyricsTab } from "@/components/AddLyricsTab";
import { EmbedLrcTab } from "@/components/EmbedLrcTab";
import { RomanizeLrcTab } from "@/components/RomanizeLrcTab";
import { ExtractLrcTab } from "@/components/ExtractLrcTab";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";

export function LyricsManagerPage() {
  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Lyrics Manager</CardTitle>
          <CardDescription>
            Add, embed, romanize, and extract lyrics for your FLAC files.
          </CardDescription>
        </CardHeader>
      </Card>

      <Tabs defaultValue="add" className="w-full">
        <TabsList className="w-full sm:w-auto">
          <TabsTrigger value="add">Add Lyrics</TabsTrigger>
          <TabsTrigger value="embed">Embed LRC</TabsTrigger>
          <TabsTrigger value="romanize">Romanize</TabsTrigger>
          <TabsTrigger value="extract">Extract</TabsTrigger>
        </TabsList>
        <TabsContent value="add" className="mt-4">
          <AddLyricsTab />
        </TabsContent>
        <TabsContent value="embed" className="mt-4">
          <EmbedLrcTab />
        </TabsContent>
        <TabsContent value="romanize" className="mt-4">
          <RomanizeLrcTab />
        </TabsContent>
        <TabsContent value="extract" className="mt-4">
          <ExtractLrcTab />
        </TabsContent>
      </Tabs>
    </div>
  );
}
