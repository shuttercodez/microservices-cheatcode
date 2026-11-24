import { ref, type Ref } from "vue";
import type {
  CreateOrderRequest,
  Order,
  PaginatedOrders,
} from "~/types/order";
import { apiRequest } from "~/utils/api-client";

export function useOrders(apiBaseUrl: Ref<string>) {
  const orders = ref<Order[]>([]);
  const page = ref(1);
  const limit = ref(10);
  const total = ref(0);
  const totalPages = ref(0);
  const loading = ref(false);

  async function fetchOrders() {
    loading.value = true;
    try {
      const res = await apiRequest<PaginatedOrders>(
        `/api/v1/orders?page=${page.value}&limit=${limit.value}`,
        { cache: "no-store" },
        apiBaseUrl.value
      );
      orders.value = res.data || [];
      total.value = res.total;
      totalPages.value = res.total_pages;
    } finally {
      loading.value = false;
    }
  }

  async function createOrder(req: CreateOrderRequest): Promise<Order> {
    const created = await apiRequest<Order>(
      "/api/v1/orders",
      {
        method: "POST",
        body: JSON.stringify(req),
      },
      apiBaseUrl.value
    );
    await fetchOrders();
    return created;
  }

  return {
    orders,
    page,
    limit,
    total,
    totalPages,
    loading,
    fetchOrders,
    createOrder,
  };
}
