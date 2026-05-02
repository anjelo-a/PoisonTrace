import { RouterProvider } from "react-router-dom";
import { QueryClientProvider } from "@tanstack/react-query";
import { router } from './routes';
import { ThemeWrapper } from './components/ThemeWrapper';
import { queryClient } from "./lib/query";

export default function App() {
  return (
    <ThemeWrapper>
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
      </QueryClientProvider>
    </ThemeWrapper>
  );
}
