<script setup lang="ts">
import { onMounted, ref, watch } from "vue";
import type { CreateOrderRequest } from "~/types/order";

const route = useRoute();
const { apiBaseUrl, isAuthenticated, showSuccess, showError } = useConsole();
const ordersState = useOrders(apiBaseUrl);
const productsState = useProducts(apiBaseUrl);

const selectedProductId = ref<string>(
  (route.query.productId as string) || ""
);

watch(
  () => route.query.productId,
  (newId) => {
    if (typeof newId === "string") {
      selectedProductId.value = newId;
    }
  }
);

async function loadOrdersAndProducts() {
  if (!isAuthenticated.value) return;
  try {
    await Promise.all([
      ordersState.fetchOrders(),
      productsState.fetchProducts(),
    ]);
  } catch (err: unknown) {
    showError(err instanceof Error ? err.message : "Failed to load orders");
  }
}

onMounted(() => {
  loadOrdersAndProducts();
});

watch([ordersState.page, apiBaseUrl], () => {
  loadOrdersAndProducts();
});

async function handleCreateOrder(req: CreateOrderRequest) {
  try {
    const created = await ordersState.createOrder(req);
    await productsState.fetchProducts();
    showSuccess(
      `Order placed (#${created.id.slice(0, 8)}) — Total: $${created.total_price.toLocaleString()}`
    );
  } catch (err: unknown) {
    showError(err instanceof Error ? err.message : "Failed to place order");
  }
}
</script>

<template>
  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <OrdersOrderForm
      v-model:selected-product-id="selectedProductId"
      :products="productsState.products.value"
      @create="handleCreateOrder"
    />
    <OrdersOrderTable
      :orders="ordersState.orders.value"
      :page="ordersState.page.value"
      :total-pages="ordersState.totalPages.value"
      :total="ordersState.total.value"
      @page-change="ordersState.page.value = $event"
    />
  </div>
</template>
