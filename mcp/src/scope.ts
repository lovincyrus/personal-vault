/**
 * MCP-layer scope enforcement — mirrors internal/vault/scope.go.
 *
 * Patterns: "*" (all), "category.*" (category), "category.field" (exact).
 */

export function getScope(): string {
  return process.env.VAULT_SCOPE ?? "*";
}

export function scopeAllows(scope: string, fieldID: string): boolean {
  for (let p of scope.split(",")) {
    p = p.trim();
    if (p === "*") return true;
    if (p.endsWith(".*")) {
      const category = p.slice(0, -2);
      if (fieldID.startsWith(category + ".")) return true;
      continue;
    }
    if (p === fieldID) return true;
  }
  return false;
}

export function scopeAllowsCategory(scope: string, category: string): boolean {
  for (let p of scope.split(",")) {
    p = p.trim();
    if (p === "*") return true;
    if (p.endsWith(".*")) {
      if (p.slice(0, -2) === category) return true;
      continue;
    }
    if (p.startsWith(category + ".")) return true;
  }
  return false;
}
