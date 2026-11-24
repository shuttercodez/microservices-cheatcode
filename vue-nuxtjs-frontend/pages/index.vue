<script setup lang="ts">
import { ref, onMounted, watch } from "vue";
import type { OAuthTokenRequest, RegisterUserRequest } from "~/types/auth";
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

const authState = useAuth(apiBaseUrl, showError);
const productsState = useProducts(apiBaseUrl);
const ordersState = useOrders(apiBaseUrl);

async function syncDashboard() {
  try {
    health.value = await checkServiceHealth(apiBaseUrl.value);
    if (authState.isAuthenticated.value) {
      await Promise.all([
        productsState.fetchProducts(),
        ordersState.fetchOrders(),
      ]);
    }
  } catch (err: unknown) {
    showError(
      err instanceof Error ? err.message : "Failed to connect to backend API"
    );
  }
}

onMounted(() => {
  syncDashboard();
});

watch(
  [authState.isAuthenticated, productsState.page, ordersState.page],
  () => {
    syncDashboard();
  }
);

async function handleLogin(req: OAuthTokenRequest) {
  try {
    const res = await authState.login(req);
    showSuccess(
      `Authenticated via OAuth 2.0 (${req.grant_type}) — Token expires in ${res.expires_in}s`
    );
    await syncDashboard();
  } catch (err: unknown) {
    showError(err instanceof Error ? err.message : "Authentication failed");
  }
}

async function handleRegister(req: RegisterUserRequest) {
  try {
    const user = await authState.register(req);
    await authState.login({
      grant_type: "password",
      username: req.username,
      password: req.password,
    });
    showSuccess(
      `Registered user "${user.username}" (${user.role}) in PostgreSQL and signed in!`
    );
    await syncDashboard();
  } catch (err: unknown) {
    showError(err instanceof Error ? err.message : "Registration failed");
  }
}

function handleLogout() {
  authState.logout();
  showSuccess("Logged out and cleared JWT token");
}

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
    await productsState.fetchProducts();
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
      :is-authenticated="authState.isAuthenticated.value"
      @refresh="syncDashboard"
      @logout="handleLogout"
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

    <AuthLoginCard
      v-if="!authState.isAuthenticated.value"
      :loading="authState.loading.value"
      :enable-register="authState.enableRegister.value"
      @login="handleLogin"
      @register="handleRegister"
    />

    <template v-else>
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
    </template>
  </div>
</template>
