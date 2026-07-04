type LyricsChoice = "original" | "romanized";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";

interface Props {
  onChoose: (choice: LyricsChoice) => void;
}

export function LyricsChoice({ onChoose }: Props) {
  return (
    <Card>
      <CardContent className="p-5 text-center">
        <img
          src="/assets/image/lymuru-found.png"
          alt="Choice"
          className="h-14 w-auto mx-auto mb-3"
        />
        <p className="text-sm font-semibold text-foreground mb-4">
          Choose lyrics format
        </p>
        <div className="flex gap-3 justify-center">
          <Button onClick={() => onChoose("original")}>Original</Button>
          <Button variant="outline" onClick={() => onChoose("romanized")}>
            Romanized
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
