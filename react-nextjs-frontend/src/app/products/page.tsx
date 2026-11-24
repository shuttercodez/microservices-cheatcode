"use client";

import React, { useEffect } from "react";
import { useRouter } from "next/navigation";
import { ProductForm } from "@/features/products/components/ProductForm";
import { ProductTable } from "@/features/products/components/ProductTable";
import { useProducts } from "@/features/products/hooks/useProducts";
import {
  CreateProductRequest,
  Product,
} from "@/features/products/types/product";
import { useConsole } from "@/providers/ConsoleProvider";

export default function ProductsPage() {
  const router = useRouter();
  const { apiBaseUrl, isAuthenticated, showSuccess, showError } = useConsole();
  const productsState = useProducts(apiBaseUrl);

  useEffect(() => {
    if (!isAuthenticated) return;
    productsState.fetchProducts().catch((err: unknown) => {
      showError(
        err instanceof Error ? err.message : "Failed to load products"
      );
    });
  }, [isAuthenticated, productsState.fetchProducts, showError]);

  if (!isAuthenticated) return null;

  const handleCreateProduct = async (req: CreateProductRequest) => {
    try {
      const created = await productsState.createProduct(req);
      showSuccess(`Created product "${created.name}" (${created.sku})`);
    } catch (err: unknown) {
      showError(
        err instanceof Error ? err.message : "Failed to create product"
      );
    }
  };

  const handlePatchStock = async (productId: string, newStock: number) => {
    try {
      const updated = await productsState.patchStock(productId, newStock);
      showSuccess(
        `Updated stock for ${updated.sku} to ${updated.stock} (v${updated.version})`
      );
    } catch (err: unknown) {
      showError(err instanceof Error ? err.message : "Failed to update stock");
    }
  };

  const handleDeleteProduct = async (product: Product) => {
    try {
      await productsState.deleteProduct(product.id);
      showSuccess(`Soft-deleted product ${product.sku}`);
    } catch (err: unknown) {
      showError(
        err instanceof Error ? err.message : "Failed to delete product"
      );
    }
  };

  return (
    <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <ProductForm onCreate={handleCreateProduct} />
      <ProductTable
        products={productsState.products}
        page={productsState.page}
        totalPages={productsState.totalPages}
        total={productsState.total}
        onPageChange={productsState.setPage}
        onPatchStock={handlePatchStock}
        onDelete={handleDeleteProduct}
        onSelectForOrder={(productId) =>
          router.push(`/orders?productId=${encodeURIComponent(productId)}`)
        }
      />
    </div>
  );
}
