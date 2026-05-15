import { useMemo } from "react";
import { useSearchParams } from "react-router-dom";

export function useUrlPagination(defaultPage = 1, defaultPageSize = 50) {
  const [params, setParams] = useSearchParams();

  const page = useMemo(() => {
    const raw = Number(params.get("page") ?? defaultPage);
    return Number.isFinite(raw) && raw > 0 ? raw : defaultPage;
  }, [defaultPage, params]);

  const pageSize = useMemo(() => {
    const raw = Number(params.get("page_size") ?? defaultPageSize);
    return Number.isFinite(raw) && raw > 0 ? raw : defaultPageSize;
  }, [defaultPageSize, params]);

  const setPage = (nextPage: number) => {
    const next = new URLSearchParams(params);
    next.set("page", String(Math.max(1, nextPage)));
    next.set("page_size", String(pageSize));
    setParams(next, { replace: false });
  };

  return { page, pageSize, setPage, params, setParams };
}
