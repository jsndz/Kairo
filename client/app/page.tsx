import { Button } from "@/components/ui/button";
import { Rocket } from "lucide-react";

export default function Home() {
  return (
    <main className="flex min-h-screen items-center justify-center bg-background">
      <Button className="flex gap-2">
        <Rocket className="h-4 w-4" /> Launch Kairo
      </Button>
    </main>
  );
}
