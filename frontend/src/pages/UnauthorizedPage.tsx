import { Button } from "@/components/common/Button";

export function UnauthorizedPage() {
  return (
    <div className="container mx-auto flex h-[calc(100vh-200px)] flex-col items-center justify-center space-y-4">
      <h1 className="h1-bold text-4xl">Unauthorized Access</h1>
      <p className="text-muted-foreground">
        You do not have permission to access this page.
      </p>
      <Button asChild to="/">
        Return Home
      </Button>
    </div>
  );
}
