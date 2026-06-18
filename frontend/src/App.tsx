import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { SessionsPage } from "@/routes/Sessions";
import { SearchPage } from "@/routes/Search";

const queryClient = new QueryClient({
  defaultOptions: { queries: { staleTime: 2000 } },
});

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter basename="/viewer">
        <Routes>
          <Route path="/" element={<SessionsPage />} />
          <Route path="/search" element={<SearchPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  );
}
