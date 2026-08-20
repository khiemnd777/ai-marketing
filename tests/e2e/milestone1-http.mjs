import assert from "node:assert/strict";

const baseURL = process.env.STUDIO_API_URL ?? "http://127.0.0.1:8080/v1";
const email = process.env.STUDIO_TEST_EMAIL;
const password = process.env.STUDIO_TEST_PASSWORD;
assert(email && password, "STUDIO_TEST_EMAIL and STUDIO_TEST_PASSWORD are required");

const login = await fetch(`${baseURL}/auth/login`, {
  method: "POST",
  headers: { "content-type": "application/json", origin: "http://localhost:3000" },
  body: JSON.stringify({ email, password }),
});
const loginText = await login.text();
assert.equal(login.status, 200, loginText);
const loginBody = JSON.parse(loginText);
const cookie = login.headers.getSetCookie().map((value) => value.split(";", 1)[0]).join("; ");
const headers = { "content-type": "application/json", origin: "http://localhost:3000", cookie, "x-csrf-token": loginBody.csrfToken };

async function request(method, path, body, idempotent = false) {
  const response = await fetch(`${baseURL}${path}`, {
    method,
    headers: { ...headers, ...(idempotent ? { "idempotency-key": crypto.randomUUID() } : {}) },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const text = await response.text();
  const data = text ? JSON.parse(text) : null;
  if (!response.ok) throw new Error(`${method} ${path}: ${response.status} ${text}`);
  return { status: response.status, data };
}

const client = (await request("POST", "/clients", { companyName: "Northstar Travel", contactName: "Demo Operator", market: "Việt Nam" }, true)).data;
const workspace = (await request("POST", `/clients/${client.id}/workspaces`, { name: "Northstar Luggage", slug: "northstar-luggage", timezone: "Asia/Ho_Chi_Minh" }, true)).data;
const brand = (await request("POST", `/clients/${client.id}/workspaces/${workspace.id}/brands`, {
  name: "Northstar", primaryLanguage: "vi", toneOfVoice: "Rõ ràng, thực tế", logoAssetIds: [], preferredTerminology: [], prohibitedTerminology: [],
}, true)).data;
const product = (await request("POST", `/clients/${client.id}/workspaces/${workspace.id}/products`, {
  brandId: brand.id, name: "Northstar Cabin 20", sku: "NS-C20-BLK", model: "C20", category: "Vali xách tay", verticalKey: "travel-luggage",
  features: [], benefits: [], differentiators: [], variants: [],
  verticalData: { luggageType: "CARRY_ON", sizeInches: 20, externalDimensions: { heightCm: 55, widthCm: 36, depthCm: 23 }, emptyWeightKg: 2.9, capacityLiters: 38, shellMaterial: "Polycarbonate", wheelType: "Spinner", wheelCount: 4, lockType: "TSA", handleType: "Telescopic", interiorCompartments: ["Divider"], expandable: true, waterResistance: "Splash resistant", warranty: "5 years", availableColors: ["Black"] },
}, true)).data;
const fact = (await request("POST", `/clients/${client.id}/workspaces/${workspace.id}/products/${product.id}/facts`, {
  factKey: "external_dimensions", label: "Kích thước ngoài", exactValue: "55 x 36 x 23 cm", normalizedValue: { heightCm: 55, widthCm: 36, depthCm: 23 }, unit: "cm", sourceName: "Approved specification sheet", sourceExcerpt: "External: 55 x 36 x 23 cm",
})).data;
const approved = (await request("POST", `/clients/${client.id}/workspaces/${workspace.id}/products/${product.id}/facts/${fact.id}/approve`, { lock: true, version: fact.version })).data;
assert.equal(approved.lockedValue, true);
assert.equal(approved.status, "APPROVED");

const lockedEdit = await fetch(`${baseURL}/clients/${client.id}/workspaces/${workspace.id}/products/${product.id}/facts/${fact.id}`, {
  method: "PUT", headers, body: JSON.stringify({ ...fact, exactValue: "56 x 36 x 23 cm", version: approved.version }),
});
assert.equal(lockedEdit.status, 409, "approved locked facts must reject edits");

const crossWorkspace = await request("POST", `/clients/${client.id}/workspaces`, { name: "Other Unit", slug: "other-unit", timezone: "UTC" }, true);
const isolated = await fetch(`${baseURL}/clients/${client.id}/workspaces/${crossWorkspace.data.id}/products/${product.id}`, { headers });
assert.equal(isolated.status, 404, "cross-workspace product reads must be hidden");

process.stdout.write(JSON.stringify({ login: 200, client: client.id, workspace: workspace.id, brand: brand.id, product: product.id, factLocked: true, crossWorkspaceStatus: 404 }) + "\n");
