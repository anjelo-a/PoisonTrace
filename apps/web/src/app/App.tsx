import { RouterProvider } from "react-router-dom";
import { router } from './routes';
import { ThemeWrapper } from './components/ThemeWrapper';

export default function App() {
  return (
    <ThemeWrapper>
      <RouterProvider router={router} />
    </ThemeWrapper>
  );
}
