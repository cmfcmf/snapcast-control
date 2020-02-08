import { useLocation } from "react-router-dom";

export function useQuery() {
  return new URLSearchParams(useLocation().search);
}

export function useIsAdmin() {
  return useQuery().get('is_admin') === "1";
}

export function times<T>(n: number, fn: (i: number) => T): T[] {
  return Array.from(new Array(n)).map((_, i) => fn(i));
}