/**
 * DMS React Query hooks — thin wrappers over `dms-api.ts`.
 *
 * Requires `@tanstack/react-query` v5 and a `QueryClientProvider` at the root.
 * Pass the platform access token (aud must include `iag.gateway` + `iag.dms`)
 * into each hook; queries are keyed by resource + params, and mutations
 * invalidate the lists that the corresponding backend event would change
 * (see FRONTEND_INTEGRATION.md §7).
 *
 *   import { useOutlets, useCreateOrder } from "@/lib/dms-react-query";
 *
 * Copy this file (and dms-api.ts) into your Next.js app under e.g. src/lib/.
 */

import {
  useQuery,
  useMutation,
  useQueryClient,
  type UseQueryOptions,
} from "@tanstack/react-query";

import { dmsApi, type Paginated } from "./dms-api";

type ListParams = Record<string, string | number | undefined>;

/** Root key so callers can `invalidateQueries({ queryKey: dmsKeys.all })`. */
export const dmsKeys = {
  all: ["dms"] as const,
  resource: (name: string, params?: ListParams) =>
    params ? (["dms", name, params] as const) : (["dms", name] as const),
};

/* ------------------------------------------------------------------ */
/* Queries                                                             */
/* ------------------------------------------------------------------ */

/**
 * Generic query helper. `enabled` is gated on the token so hooks are safe to
 * call before login (they simply stay idle until a token is present).
 */
function useDmsQuery<T>(
  name: string,
  fetcher: () => Promise<T>,
  token: string | undefined,
  params?: ListParams,
  options?: Partial<UseQueryOptions<T>>,
) {
  return useQuery<T>({
    queryKey: dmsKeys.resource(name, params),
    queryFn: fetcher,
    enabled: Boolean(token) && (options?.enabled ?? true),
    ...options,
  });
}

// --- platform / session ---
export const useBootstrap = (token?: string) =>
  useDmsQuery("bootstrap", () => dmsApi.bootstrap(token), token);
export const useSession = (token?: string) =>
  useDmsQuery("session", () => dmsApi.session(token), token);
export const usePermissionsMe = (token?: string) =>
  useDmsQuery("permissions/me", () => dmsApi.permissionsMe(token), token);
export const useNotifications = (token?: string) =>
  useDmsQuery("notifications", () => dmsApi.notifications(token), token);
export const useSearch = (q: string, token?: string, limit = 18) =>
  useDmsQuery("search", () => dmsApi.search(q, token, limit), token, { q, limit }, {
    enabled: Boolean(token) && q.length > 0,
  });

// --- overview & network ---
export const useOverview = (token?: string) =>
  useDmsQuery("overview", () => dmsApi.overview(token), token);
export const useDistributors = (token?: string, params?: ListParams) =>
  useDmsQuery<Paginated<unknown>>("distributors", () => dmsApi.distributors(token, params), token, params);

// --- outlets ---
export const useOutlets = (token?: string, params?: ListParams) =>
  useDmsQuery<Paginated<unknown>>("outlets", () => dmsApi.outlets(token, params), token, params);
export const useOutletStats = (token?: string) =>
  useDmsQuery("outlets/stats", () => dmsApi.outletStats(token), token);

// --- orders ---
export const useOrders = (token?: string, params?: ListParams) =>
  useDmsQuery<Paginated<unknown>>("orders", () => dmsApi.orders(token, params), token, params);
export const useOrdersBoard = (token?: string) =>
  useDmsQuery("orders/board", () => dmsApi.ordersBoard(token), token);

// --- routes / field ---
export const useBeats = (token?: string, params?: ListParams) =>
  useDmsQuery<Paginated<unknown>>("beats", () => dmsApi.beats(token, params), token, params);
export const useCheckIns = (token?: string, params?: ListParams) =>
  useDmsQuery<Paginated<unknown>>("field/check-ins", () => dmsApi.checkIns(token, params), token, params);
export const useVisitReports = (token?: string, params?: ListParams) =>
  useDmsQuery<Paginated<unknown>>("field/visit-reports", () => dmsApi.visitReports(token, params), token, params);

// --- trade ---
export const usePromotions = (token?: string, params?: ListParams) =>
  useDmsQuery<Paginated<unknown>>("promotions", () => dmsApi.promotions(token, params), token, params);
export const useClaims = (token?: string, params?: ListParams) =>
  useDmsQuery<Paginated<unknown>>("claims", () => dmsApi.claims(token, params), token, params);
export const useDispatch = (token?: string, params?: ListParams) =>
  useDmsQuery<Paginated<unknown>>("dispatch", () => dmsApi.dispatch(token, params), token, params);

// --- finance ---
export const useFinanceSummary = (token?: string) =>
  useDmsQuery("finance/summary", () => dmsApi.financeSummary(token), token);
export const useInvoices = (token?: string, params?: ListParams) =>
  useDmsQuery<Paginated<unknown>>("invoices", () => dmsApi.invoices(token, params), token, params);

// --- intelligence ---
export const useKpiBoard = (token?: string) =>
  useDmsQuery("kpi/board", () => dmsApi.kpiBoard(token), token);
export const useAnalytics = (token?: string) =>
  useDmsQuery("analytics", () => dmsApi.analytics(token), token);
export const useForecast = (token?: string) =>
  useDmsQuery("forecast", () => dmsApi.forecast(token), token);
export const useInsightsSignals = (token?: string) =>
  useDmsQuery("insights/signals", () => dmsApi.insightsSignals(token), token);

/* ------------------------------------------------------------------ */
/* Mutations — each invalidates the lists its backend event changes    */
/* ------------------------------------------------------------------ */

function useDmsMutation<TVars>(
  fn: (vars: TVars) => Promise<unknown>,
  invalidate: string[],
) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: fn,
    onSuccess: () =>
      Promise.all(
        invalidate.map((name) => qc.invalidateQueries({ queryKey: dmsKeys.resource(name) })),
      ),
  });
}

export const useCreateOutlet = (token?: string) =>
  useDmsMutation((body: unknown) => dmsApi.createOutlet(body, token), ["outlets", "outlets/stats"]);

export const usePatchOutlet = (token?: string) =>
  useDmsMutation(
    (vars: { id: string; body: unknown }) => dmsApi.patchOutlet(vars.id, vars.body, token),
    ["outlets"],
  );

export const useCreateOrder = (token?: string) =>
  useDmsMutation((body: unknown) => dmsApi.createOrder(body, token), ["orders", "orders/board"]);

export const usePatchOrderStatus = (token?: string) =>
  useDmsMutation(
    (vars: { id: string; status: string }) => dmsApi.patchOrderStatus(vars.id, vars.status, token),
    ["orders", "orders/board"],
  );

export const useCreateCheckIn = (token?: string) =>
  useDmsMutation((body: unknown) => dmsApi.createCheckIn(body, token), ["field/check-ins"]);

export const useCompleteCheckIn = (token?: string) =>
  useDmsMutation((id: string) => dmsApi.completeCheckIn(id, token), ["field/check-ins"]);

export const useCreateVisitReport = (token?: string) =>
  useDmsMutation((body: unknown) => dmsApi.createVisitReport(body, token), ["field/visit-reports"]);

export const useCreatePromotion = (token?: string) =>
  useDmsMutation((body: unknown) => dmsApi.createPromotion(body, token), ["promotions"]);

export const useCreateClaim = (token?: string) =>
  useDmsMutation((body: unknown) => dmsApi.createClaim(body, token), ["claims"]);

export const useCreateDispatch = (token?: string) =>
  useDmsMutation((body: unknown) => dmsApi.createDispatch(body, token), ["dispatch", "orders/board"]);

export const useCreateInvoice = (token?: string) =>
  // Finance assigns the id on create, so refetch the list rather than trusting
  // a local insert — see FRONTEND_INTEGRATION.md §6.
  useDmsMutation((body: unknown) => dmsApi.createInvoice(body, token), ["invoices", "finance/summary"]);

export const useRunReport = (token?: string) =>
  useMutation({ mutationFn: (body: unknown) => dmsApi.runReport(body, token) });

export const useExportPage = (token?: string) =>
  useMutation({
    mutationFn: (vars: { page: string; body?: unknown }) =>
      dmsApi.exportPage(vars.page, vars.body ?? {}, token),
  });
