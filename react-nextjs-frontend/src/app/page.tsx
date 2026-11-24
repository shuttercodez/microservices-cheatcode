"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useConsole } from "@/providers/ConsoleProvider";

export default function HomePage() {
  const router = useRouter();
  const { authReady, isAuthenticated } = useConsole();

  useEffect(() => {
    if (!authReady) return;
    router.replace(isAuthenticated ? "/products" : "/login");
  }, [authReady, isAuthenticated, router]);

  return null;
}
