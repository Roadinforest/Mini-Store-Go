export function NotFoundPage() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center">
      <img src="/images/logo.svg" alt="Mini Store logo" className="h-12 w-12" />
      <div className="w-full max-w-md rounded-lg p-6 text-center shadow-md">
        <h1 className="mb-4 text-3xl font-bold">Not Found</h1>
        <p className="text-destructive">Could not find requested page</p>
        <button
          className="mt-4 ml-2 rounded-md border px-4 py-2 text-sm"
          onClick={() => {
            window.location.href = "/";
          }}
        >
          Back to home
        </button>
      </div>
    </div>
  );
}
