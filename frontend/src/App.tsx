import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { SessionsPage } from "@/routes/Sessions";
import { SearchPage } from "@/routes/Search";
import { LaunchPage } from "@/routes/Launch";
import { CasesPage } from "@/routes/Cases";

const queryClient = new QueryClient({
  defaultOptions: { queries: { staleTime: 2000 } },
});

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter basename="/viewer">
        <Routes>
          <Route path="/sessions" element={<SessionsPage />} />
          <Route path="/sessions/:id" element={<SessionsPage />} />
          <Route path="/launch" element={<LaunchPage />} />
          <Route path="/cases" element={<CasesPage />} />
          <Route path="/search" element={<SearchPage />} />
          <Route path="*" element={<Navigate to="/sessions" replace />} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  );
}
