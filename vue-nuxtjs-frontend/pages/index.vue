<script setup lang="ts">
import { ref, onMounted, watch } from "vue";
import type { HealthStatus } from "~/types/common";
import type { CreateOrderRequest } from "~/types/order";
import type { CreateProductRequest, Product } from "~/types/product";
import { checkServiceHealth, DEFAULT_API_BASE_URL } from "~/utils/api-client";

const config = useRuntimeConfig();

const apiBaseUrl = ref<string>(
  (config.public.apiBaseUrl as string) || DEFAULT_API_BASE_URL
);
const health = ref<HealthStatus>({ alive: false, ready: false });
const activeTab = ref<"products" | "orders">("products");
const selectedProductId = ref("");

const errorMsg = ref<string | null>(null);
const successMsg = ref<string | null>(null);

const productsState = useProducts(apiBaseUrl);
const ordersState = useOrders(apiBaseUrl);

function showSuccess(msg: string) {
  successMsg.value = msg;
  errorMsg.value = null;
  setTimeout(() => {
    successMsg.value = null;
  }, 4000);
}

function showError(msg: string) {
  errorMsg.value = msg;
  successMsg.value = null;
}

async function syncDashboard() {
  try {
    health.value = await checkServiceHealth(apiBaseUrl.value);
    await Promise.all([
      productsState.fetchProducts(),
      ordersState.fetchOrders(),
    ]);
  } catch (err: unknown) {
    showError(
      err instanceof Error ? err.message : "Failed to connect to backend API"
    );
  }
}

onMounted(() => {
  syncDashboard();
});

watch([productsState.page, ordersState.page], () => {
  syncDashboard();
});

async function handleCreateProduct(req: CreateProductRequest) {
  try {
    const created = await productsState.createProduct(req);
    showSuccess(`Created product "${created.name}" (${created.sku})`);
  } catch (err: unknown) {
    showError(err instanceof Error ? err.message : "Failed to create product");
  }
}

async function handlePatchStock(productId: string, newStock: number) {
  try {
    const updated = await productsState.patchStock(productId, newStock);
    showSuccess(
      `Updated stock for ${updated.sku} to ${updated.stock} (v${updated.version})`
    );
  } catch (err: unknown) {
    showError(err instanceof Error ? err.message : "Failed to update stock");
  }
}

async function handleDeleteProduct(product: Product) {
  try {
    await productsState.deleteProduct(product.id);
    showSuccess(`Soft-deleted product ${product.sku}`);
  } catch (err: unknown) {
    showError(err instanceof Error ? err.message : "Failed to delete product");
  }
}

async function handleCreateOrder(req: CreateOrderRequest) {
  try {
    const created = await ordersState.createOrder(req);
    await productsState.fetchProducts(); // Refresh product stock after deduction
    showSuccess(
      `Order placed (#${created.id.slice(0, 8)}) — Total: $${created.total_price.toLocaleString()}`
    );
  } catch (err: unknown) {
    showError(err instanceof Error ? err.message : "Failed to place order");
  }
}

function handleSelectForOrder(productId: string) {
  selectedProductId.value = productId;
  activeTab.value = "orders";
}
</script>

<template>
  <div class="max-w-6xl mx-auto px-4 py-8">
    <LayoutNavbar
      v-model:api-base-url="apiBaseUrl"
      :health="health"
      :loading="productsState.loading.value || ordersState.loading.value"
      @refresh="syncDashboard"
    />

    <UiAlertBanner
      type="error"
      :message="errorMsg"
      @dismiss="errorMsg = null"
    />
    <UiAlertBanner
      type="success"
      :message="successMsg"
      @dismiss="successMsg = null"
    />

    <!-- Navigation Tabs -->
    <div class="flex gap-2 mb-6 border-b border-slate-200">
      <button
        :class="[
          'px-5 py-2.5 text-sm font-semibold border-b-2 -mb-px transition',
          activeTab === 'products'
            ? 'border-emerald-600 text-emerald-600'
            : 'border-transparent text-slate-600 hover:text-slate-900',
        ]"
        @click="activeTab = 'products'"
      >
        📦 Products ({{ productsState.total.value }})
      </button>
      <button
        :class="[
          'px-5 py-2.5 text-sm font-semibold border-b-2 -mb-px transition',
          activeTab === 'orders'
            ? 'border-emerald-600 text-emerald-600'
            : 'border-transparent text-slate-600 hover:text-slate-900',
        ]"
        @click="activeTab = 'orders'"
      >
        🧾 Orders ({{ ordersState.total.value }})
      </button>
    </div>

    <!-- PRODUCTS TAB -->
    <div
      v-if="activeTab === 'products'"
      class="grid grid-cols-1 lg:grid-cols-3 gap-6"
    >
      <ProductsProductForm @create="handleCreateProduct" />
      <ProductsProductTable
        :products="productsState.products.value"
        :page="productsState.page.value"
        :total-pages="productsState.totalPages.value"
        :total="productsState.total.value"
        @page-change="productsState.page.value = $event"
        @patch-stock="handlePatchStock"
        @delete="handleDeleteProduct"
        @select-for-order="handleSelectForOrder"
      />
    </div>

    <!-- ORDERS TAB -->
    <div
      v-if="activeTab === 'orders'"
      class="grid grid-cols-1 lg:grid-cols-3 gap-6"
    >
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
  </div>
</template>
