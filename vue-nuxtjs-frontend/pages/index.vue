<script setup lang="ts">
import { ref, onMounted, watch } from "vue";
import {
  useMicroserviceApi,
  type Product,
  type Order,
} from "~/composables/useMicroserviceApi";

const config = useRuntimeConfig();
const api = useMicroserviceApi();

const apiBaseUrl = ref<string>(
  (config.public.apiBaseUrl as string) || "http://localhost:8080"
);
const health = ref({ alive: false, ready: false });
const activeTab = ref<"products" | "orders">("products");

// Products state
const products = ref<Product[]>([]);
const prodPage = ref(1);
const prodLimit = ref(10);
const prodTotal = ref(0);
const prodTotalPages = ref(0);

// Create Product form
const sku = ref("");
const name = ref("");
const price = ref<number | string>("");
const stock = ref<number | string>("");

// Inline stock patch state
const editingStockId = ref<string | null>(null);
const newStockValue = ref<number | string>("");

// Orders state
const orders = ref<Order[]>([]);
const ordPage = ref(1);
const ordLimit = ref(10);
const ordTotal = ref(0);
const ordTotalPages = ref(0);

// Create Order form
const selectedProductId = ref("");
const orderQuantity = ref<number | string>(1);

// Status alerts
const errorMsg = ref<string | null>(null);
const successMsg = ref<string | null>(null);
const loading = ref(false);

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

async function fetchAll() {
  loading.value = true;
  try {
    health.value = await api.checkHealth(apiBaseUrl.value);

    const [prodRes, ordRes] = await Promise.all([
      api.getProducts(prodPage.value, prodLimit.value, apiBaseUrl.value),
      api.getOrders(ordPage.value, ordLimit.value, apiBaseUrl.value),
    ]);

    products.value = prodRes.data || [];
    prodTotal.value = prodRes.total;
    prodTotalPages.value = prodRes.total_pages;

    orders.value = ordRes.data || [];
    ordTotal.value = ordRes.total;
    ordTotalPages.value = ordRes.total_pages;
  } catch (err: unknown) {
    showError(
      err instanceof Error ? err.message : "Failed to connect to backend API"
    );
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  fetchAll();
});

watch([prodPage, ordPage], () => {
  fetchAll();
});

async function handleCreateProduct() {
  try {
    const created = await api.createProduct(
      {
        sku: sku.value.trim(),
        name: name.value.trim(),
        price: Number(price.value) || 0,
        stock: Number(stock.value) || 0,
      },
      apiBaseUrl.value
    );
    sku.value = "";
    name.value = "";
    price.value = "";
    stock.value = "";
    showSuccess(`Created product "${created.name}" (${created.sku})`);
    await fetchAll();
  } catch (err: unknown) {
    showError(err instanceof Error ? err.message : "Failed to create product");
  }
}

async function handlePatchStock(productId: string) {
  try {
    const parsed = Number(newStockValue.value);
    if (Number.isNaN(parsed)) return;
    const updated = await api.patchProductStock(
      productId,
      parsed,
      apiBaseUrl.value
    );
    editingStockId.value = null;
    showSuccess(
      `Updated stock for ${updated.sku} to ${updated.stock} (v${updated.version})`
    );
    await fetchAll();
  } catch (err: unknown) {
    showError(err instanceof Error ? err.message : "Failed to update stock");
  }
}

async function handleDeleteProduct(p: Product) {
  try {
    await api.deleteProduct(p.id, apiBaseUrl.value);
    showSuccess(`Soft-deleted product ${p.sku}`);
    await fetchAll();
  } catch (err: unknown) {
    showError(err instanceof Error ? err.message : "Failed to delete product");
  }
}

async function handleCreateOrder() {
  try {
    const created = await api.createOrder(
      {
        product_id: selectedProductId.value.trim(),
        quantity: Number(orderQuantity.value) || 1,
      },
      apiBaseUrl.value
    );
    orderQuantity.value = 1;
    showSuccess(
      `Order placed (#${created.id.slice(0, 8)}) — Total: $${created.total_price.toLocaleString()}`
    );
    await fetchAll();
  } catch (err: unknown) {
    showError(err instanceof Error ? err.message : "Failed to place order");
  }
}
</script>

<template>
  <div class="max-w-6xl mx-auto px-4 py-8">
    <!-- Header & Health Status -->
    <header
      class="bg-white rounded-xl shadow-sm border border-slate-200 p-6 mb-6 flex flex-col md:flex-row md:items-center md:justify-between gap-4"
    >
      <div>
        <div class="flex items-center gap-2">
          <span
            class="px-2.5 py-0.5 text-xs font-semibold bg-emerald-100 text-emerald-800 rounded-full"
          >
            Vue 3 + Nuxt 3
          </span>
          <span
            :class="[
              'px-2.5 py-0.5 text-xs font-semibold rounded-full',
              health.alive
                ? 'bg-emerald-100 text-emerald-800'
                : 'bg-rose-100 text-rose-800',
            ]"
          >
            /healthz: {{ health.alive ? "ALIVE" : "OFFLINE" }}
          </span>
          <span
            :class="[
              'px-2.5 py-0.5 text-xs font-semibold rounded-full',
              health.ready
                ? 'bg-emerald-100 text-emerald-800'
                : 'bg-amber-100 text-amber-800',
            ]"
          >
            /readyz: {{ health.ready ? "READY" : "NOT READY" }}
          </span>
        </div>
        <h1 class="text-2xl font-bold text-slate-900 mt-2">
          Inventory &amp; Order Microservice Console
        </h1>
        <p class="text-sm text-slate-500">
          Connects seamlessly to Go (GORM), Go (SQLx), Rust (Axum), or Python
          (FastAPI) backends
        </p>
      </div>

      <div class="flex items-center gap-2">
        <input
          v-model="apiBaseUrl"
          type="text"
          class="text-sm border border-slate-300 rounded-lg px-3 py-2 w-60 focus:outline-none focus:ring-2 focus:ring-emerald-500"
          placeholder="http://localhost:8080"
        />
        <button
          :disabled="loading"
          class="bg-slate-900 hover:bg-slate-800 text-white text-sm font-medium px-4 py-2 rounded-lg transition"
          @click="fetchAll"
        >
          {{ loading ? "Syncing..." : "Refresh" }}
        </button>
      </div>
    </header>

    <!-- Alerts -->
    <div
      v-if="errorMsg"
      class="mb-6 bg-rose-50 border border-rose-200 text-rose-700 px-4 py-3 rounded-lg text-sm flex justify-between items-center"
    >
      <span>⚠️ {{ errorMsg }}</span>
      <button
        class="text-rose-500 hover:text-rose-800 font-bold"
        @click="errorMsg = null"
      >
        ✕
      </button>
    </div>
    <div
      v-if="successMsg"
      class="mb-6 bg-emerald-50 border border-emerald-200 text-emerald-800 px-4 py-3 rounded-lg text-sm flex justify-between items-center"
    >
      <span>✅ {{ successMsg }}</span>
      <button
        class="text-emerald-600 hover:text-emerald-900 font-bold"
        @click="successMsg = null"
      >
        ✕
      </button>
    </div>

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
        📦 Products ({{ prodTotal }})
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
        🧾 Orders ({{ ordTotal }})
      </button>
    </div>

    <!-- PRODUCTS TAB -->
    <div
      v-if="activeTab === 'products'"
      class="grid grid-cols-1 lg:grid-cols-3 gap-6"
    >
      <!-- Create Product Card -->
      <div
        class="bg-white rounded-xl shadow-sm border border-slate-200 p-6 h-fit"
      >
        <h2 class="text-lg font-semibold text-slate-900 mb-4">
          Create New Product
        </h2>
        <form class="space-y-4" @submit.prevent="handleCreateProduct">
          <div>
            <label class="block text-xs font-medium text-slate-700 mb-1">
              SKU (Unique Code)
            </label>
            <input
              v-model="sku"
              type="text"
              required
              placeholder="e.g. IPHONE-16-PRO"
              class="w-full text-sm border border-slate-300 rounded-lg px-3 py-2"
            />
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-700 mb-1">
              Product Name
            </label>
            <input
              v-model="name"
              type="text"
              required
              placeholder="e.g. iPhone 16 Pro 256GB"
              class="w-full text-sm border border-slate-300 rounded-lg px-3 py-2"
            />
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-xs font-medium text-slate-700 mb-1">
                Price
              </label>
              <input
                v-model="price"
                type="number"
                step="0.01"
                min="0"
                required
                placeholder="39900"
                class="w-full text-sm border border-slate-300 rounded-lg px-3 py-2"
              />
            </div>
            <div>
              <label class="block text-xs font-medium text-slate-700 mb-1">
                Initial Stock
              </label>
              <input
                v-model="stock"
                type="number"
                min="0"
                required
                placeholder="50"
                class="w-full text-sm border border-slate-300 rounded-lg px-3 py-2"
              />
            </div>
          </div>
          <button
            type="submit"
            class="w-full bg-emerald-600 hover:bg-emerald-700 text-white text-sm font-medium py-2.5 rounded-lg transition"
          >
            + Add Product
          </button>
        </form>
      </div>

      <!-- Paginated Product Table -->
      <div
        class="lg:col-span-2 bg-white rounded-xl shadow-sm border border-slate-200 overflow-hidden"
      >
        <div
          class="px-6 py-4 border-b border-slate-200 flex justify-between items-center"
        >
          <h2 class="text-lg font-semibold text-slate-900">
            Product Inventory (Redis Cached)
          </h2>
          <span class="text-xs text-slate-500">
            Page {{ prodPage }} of {{ prodTotalPages || 1 }} ({{ prodTotal }}
            items)
          </span>
        </div>

        <div class="overflow-x-auto">
          <table class="w-full text-left text-sm">
            <thead
              class="bg-slate-50 text-xs uppercase text-slate-500 border-b border-slate-200"
            >
              <tr>
                <th class="px-6 py-3">SKU / Name</th>
                <th class="px-4 py-3">Price</th>
                <th class="px-4 py-3">Stock</th>
                <th class="px-4 py-3">Version</th>
                <th class="px-6 py-3 text-right">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-200">
              <tr v-if="products.length === 0">
                <td colspan="5" class="px-6 py-8 text-center text-slate-400">
                  No products found. Create your first product on the left.
                </td>
              </tr>
              <tr
                v-for="p in products"
                :key="p.id"
                class="hover:bg-slate-50"
              >
                <td class="px-6 py-3">
                  <div class="font-semibold text-slate-900">{{ p.sku }}</div>
                  <div class="text-xs text-slate-500">{{ p.name }}</div>
                  <div class="text-[11px] text-slate-400 font-mono">
                    {{ p.id }}
                  </div>
                </td>
                <td class="px-4 py-3 font-medium">
                  ${{ p.price.toLocaleString() }}
                </td>
                <td class="px-4 py-3">
                  <div
                    v-if="editingStockId === p.id"
                    class="flex items-center gap-1"
                  >
                    <input
                      v-model="newStockValue"
                      type="number"
                      min="0"
                      class="w-20 border border-slate-300 rounded px-2 py-1 text-xs"
                    />
                    <button
                      class="bg-emerald-600 text-white text-xs px-2 py-1 rounded"
                      @click="handlePatchStock(p.id)"
                    >
                      Save
                    </button>
                    <button
                      class="text-slate-400 text-xs px-1"
                      @click="editingStockId = null"
                    >
                      ✕
                    </button>
                  </div>
                  <span
                    v-else
                    :class="[
                      'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium',
                      p.stock > 5
                        ? 'bg-emerald-100 text-emerald-800'
                        : p.stock > 0
                        ? 'bg-amber-100 text-amber-800'
                        : 'bg-rose-100 text-rose-800',
                    ]"
                  >
                    {{ p.stock }} in stock
                  </span>
                </td>
                <td class="px-4 py-3 text-xs text-slate-500 font-mono">
                  v{{ p.version }}
                </td>
                <td class="px-6 py-3 text-right space-x-2">
                  <button
                    class="text-xs text-blue-600 hover:text-blue-800 font-medium"
                    @click="
                      editingStockId = p.id;
                      newStockValue = p.stock;
                    "
                  >
                    Patch Stock
                  </button>
                  <button
                    class="text-xs text-emerald-600 hover:text-emerald-800 font-medium"
                    @click="
                      selectedProductId = p.id;
                      activeTab = 'orders';
                    "
                  >
                    Order
                  </button>
                  <button
                    class="text-xs text-rose-600 hover:text-rose-800 font-medium"
                    @click="handleDeleteProduct(p)"
                  >
                    Delete
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Pagination Footer -->
        <div
          class="px-6 py-4 bg-slate-50 border-t border-slate-200 flex justify-between items-center"
        >
          <button
            :disabled="prodPage <= 1"
            class="px-3 py-1.5 text-xs font-medium bg-white border border-slate-300 rounded-lg disabled:opacity-40"
            @click="prodPage = Math.max(1, prodPage - 1)"
          >
            ← Previous
          </button>
          <span class="text-xs text-slate-600">
            Page {{ prodPage }} of {{ prodTotalPages || 1 }}
          </span>
          <button
            :disabled="prodPage >= prodTotalPages"
            class="px-3 py-1.5 text-xs font-medium bg-white border border-slate-300 rounded-lg disabled:opacity-40"
            @click="prodPage = prodPage + 1"
          >
            Next →
          </button>
        </div>
      </div>
    </div>

    <!-- ORDERS TAB -->
    <div
      v-if="activeTab === 'orders'"
      class="grid grid-cols-1 lg:grid-cols-3 gap-6"
    >
      <!-- Create Order Card -->
      <div
        class="bg-white rounded-xl shadow-sm border border-slate-200 p-6 h-fit"
      >
        <h2 class="text-lg font-semibold text-slate-900 mb-4">
          Place Customer Order
        </h2>
        <form class="space-y-4" @submit.prevent="handleCreateOrder">
          <div>
            <label class="block text-xs font-medium text-slate-700 mb-1">
              Select Product (or enter UUID)
            </label>
            <select
              v-model="selectedProductId"
              class="w-full text-sm border border-slate-300 rounded-lg px-3 py-2 mb-2 bg-white"
            >
              <option value="">-- Choose from current page --</option>
              <option v-for="p in products" :key="p.id" :value="p.id">
                {{ p.sku }} — {{ p.name }} (${{ p.price }}, Stock:
                {{ p.stock }})
              </option>
            </select>
            <input
              v-model="selectedProductId"
              type="text"
              required
              placeholder="Product UUID"
              class="w-full text-xs font-mono border border-slate-300 rounded-lg px-3 py-2"
            />
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-700 mb-1">
              Quantity
            </label>
            <input
              v-model="orderQuantity"
              type="number"
              min="1"
              required
              class="w-full text-sm border border-slate-300 rounded-lg px-3 py-2"
            />
          </div>
          <button
            type="submit"
            class="w-full bg-emerald-600 hover:bg-emerald-700 text-white text-sm font-medium py-2.5 rounded-lg transition"
          >
            Place Order &amp; Deduct Stock
          </button>
        </form>
      </div>

      <!-- Paginated Order Table -->
      <div
        class="lg:col-span-2 bg-white rounded-xl shadow-sm border border-slate-200 overflow-hidden"
      >
        <div
          class="px-6 py-4 border-b border-slate-200 flex justify-between items-center"
        >
          <h2 class="text-lg font-semibold text-slate-900">
            Customer Orders (Redis Cached)
          </h2>
          <span class="text-xs text-slate-500">
            Page {{ ordPage }} of {{ ordTotalPages || 1 }} ({{ ordTotal }}
            orders)
          </span>
        </div>

        <div class="overflow-x-auto">
          <table class="w-full text-left text-sm">
            <thead
              class="bg-slate-50 text-xs uppercase text-slate-500 border-b border-slate-200"
            >
              <tr>
                <th class="px-6 py-3">Order ID</th>
                <th class="px-4 py-3">Product ID</th>
                <th class="px-4 py-3">Qty</th>
                <th class="px-4 py-3">Total Price</th>
                <th class="px-6 py-3">Status</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-200">
              <tr v-if="orders.length === 0">
                <td colspan="5" class="px-6 py-8 text-center text-slate-400">
                  No orders placed yet.
                </td>
              </tr>
              <tr v-for="o in orders" :key="o.id" class="hover:bg-slate-50">
                <td class="px-6 py-3 font-mono text-xs text-slate-700">
                  {{ o.id }}
                </td>
                <td class="px-4 py-3 font-mono text-xs text-slate-500">
                  {{ o.product_id }}
                </td>
                <td class="px-4 py-3 font-semibold">{{ o.quantity }}</td>
                <td class="px-4 py-3 font-semibold text-emerald-700">
                  ${{ o.total_price.toLocaleString() }}
                </td>
                <td class="px-6 py-3">
                  <span
                    class="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-emerald-100 text-emerald-800"
                  >
                    {{ o.status }}
                  </span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Pagination Footer -->
        <div
          class="px-6 py-4 bg-slate-50 border-t border-slate-200 flex justify-between items-center"
        >
          <button
            :disabled="ordPage <= 1"
            class="px-3 py-1.5 text-xs font-medium bg-white border border-slate-300 rounded-lg disabled:opacity-40"
            @click="ordPage = Math.max(1, ordPage - 1)"
          >
            ← Previous
          </button>
          <span class="text-xs text-slate-600">
            Page {{ ordPage }} of {{ ordTotalPages || 1 }}
          </span>
          <button
            :disabled="ordPage >= ordTotalPages"
            class="px-3 py-1.5 text-xs font-medium bg-white border border-slate-300 rounded-lg disabled:opacity-40"
            @click="ordPage = ordPage + 1"
          >
            Next →
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
