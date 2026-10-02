import { parse, TYPE, type MessageFormatElement } from "@formatjs/icu-messageformat-parser";
import type { CatalogMessages } from "@/i18n/catalogs";

function argumentsIn(elements: MessageFormatElement[], result = new Set<string>()) {
  for (const element of elements) {
    if (element.type !== TYPE.literal && element.type !== TYPE.pound)
      result.add(`${element.type}:${element.value}`);
    if (element.type === TYPE.select || element.type === TYPE.plural) {
      for (const option of Object.values(element.options)) argumentsIn(option.value, result);
    }
    if (element.type === TYPE.tag) argumentsIn(element.children, result);
  }
  return [...result].sort().join(",");
}

function leaves(messages: CatalogMessages, prefix = "", result = new Map<string, string>()) {
  for (const [key, value] of Object.entries(messages)) {
    const path = prefix ? `${prefix}.${key}` : key;
    if (typeof value === "string")
      result.set(path, argumentsIn(parse(value, { requiresOtherClause: true })));
    else leaves(value, path, result);
  }
  return result;
}

export function validateCatalogs(catalogs: CatalogMessages[]) {
  const baseline = leaves(catalogs[0]);
  for (const catalog of catalogs.slice(1)) {
    const candidate = leaves(catalog);
    if (candidate.size !== baseline.size) throw new Error("Catalog key count differs");
    for (const [key, argumentsList] of baseline) {
      if (candidate.get(key) !== argumentsList)
        throw new Error(`Catalog key/arguments differ: ${key}`);
    }
  }
}
