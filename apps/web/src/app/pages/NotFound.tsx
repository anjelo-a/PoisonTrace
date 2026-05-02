import { Link } from "react-router-dom";
import { AlertCircle } from "lucide-react";

export default function NotFound() {
  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-8">
      <div className="max-w-md w-full text-center">
        <AlertCircle className="w-16 h-16 mx-auto mb-8 text-muted-foreground" />
        <h1 className="text-6xl font-mono mb-6 tracking-tight">404</h1>
        <p className="text-lg mb-3">Page Not Found</p>
        <p className="text-sm text-muted-foreground mb-12 leading-relaxed">
          The page you're looking for doesn't exist or has been moved.
        </p>
        <div className="flex gap-6 justify-center">
          <Link
            to="/"
            className="px-8 py-3 border border-border hover:border-foreground hover:text-foreground transition-colors"
          >
            Go Home
          </Link>
          <Link
            to="/app"
            className="px-8 py-3 bg-foreground text-background hover:bg-muted-foreground transition-colors"
          >
            Go to App
          </Link>
        </div>
      </div>
    </div>
  );
}
