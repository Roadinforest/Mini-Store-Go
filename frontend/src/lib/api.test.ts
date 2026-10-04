import { afterEach, describe, expect, it, vi } from "vitest";
import * as api from "./api";

afterEach(() => vi.unstubAllGlobals());
const envelope = (data: unknown, status = 200) => new Response(JSON.stringify({code: status === 200 ? "OK" : "UNAUTHORIZED", message: "response", data}), {status, headers: {"Content-Type":"application/json"}});
const product = {id:"p1", name:"Phone", slug:"phone", category:"phone", images:null, brand:"Brand", description:"Phone", stock:3, price:12.5, rating:4.5, num_reviews:2, is_featured:true, created_at:"2026-01-01"};
const meta = {page:1, limit:20, total:0, total_pages:0};

describe("endpoint response contracts", () => {
  it("decodes products and normalizes a nullable image array", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(envelope(product)));
    const result = await api.getProductByID("p1");
    expect(result.data).toMatchObject({numReviews:2, isFeatured:true, price:12.5, images:[]});
    expect(result.data).not.toHaveProperty("num_reviews");
  });
  it.each([api.getProducts, api.getAdminUsers, api.getMyOrders])("decodes empty pages using the endpoint contract", async getPage => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(envelope({items:[], meta})));
    expect((await getPage()).data).toEqual({items:[],meta:{page:1,limit:20,total:0,totalPages:0}});
  });
  it("rejects wrong endpoint data and malformed numeric fields", async () => {
    const fetch = vi.fn().mockResolvedValueOnce(envelope({id:"u1", email:"test@test.invalid"})).mockResolvedValueOnce(envelope({...product, price:"12.50"}));
    vi.stubGlobal("fetch", fetch);
    for (let i=0;i<2;i++) expect(await api.getProductByID("p1")).toMatchObject({success:false, message:"Invalid API response."});
  });
  it("decodes order items, shipping fields and nullable timestamps", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(envelope({
      id:"o1",user_id:"u1",shipping_address:{full_name:"User",street_address:"Street",city:"City",postal_code:"123",country:"CN"},
      payment_method:"cash",items_price:12,shipping_price:10,tax_price:1.8,total_price:23.8,is_paid:false,is_delivered:false,created_at:"2026-01-01",
      order_items:[{product_id:"p1",name:"Phone",slug:"phone",qty:1,image:"",price:12}],
    })));
    expect((await api.getOrderByID("o1")).data).toMatchObject({shippingAddress:{fullName:"User"},orderitems:[{productId:"p1",qty:1}],paidAt:null,deliveredAt:null});
  });
  it("coalesces concurrent refresh requests and retries each request only once", async () => {
    let release!: () => void;
    const gate = new Promise<void>(resolve => {release=resolve;});
    const fetch = vi.fn(async (url: string) => {
      if (url.endsWith("/auth/refresh")) { await gate; return envelope({id:"u1",name:"User",email:"user@test.invalid",role:"user"}); }
      return envelope(undefined,401);
    });
    vi.stubGlobal("fetch", fetch);
    const requests = [api.getProducts(),api.getAdminUsers()];
    await vi.waitFor(() => expect(fetch.mock.calls.filter(([url]) => url.endsWith("/auth/refresh"))).toHaveLength(1));
    release();
    expect((await Promise.all(requests)).every(result => !result.success)).toBe(true);
    expect(fetch.mock.calls).toHaveLength(5);
  });
});
