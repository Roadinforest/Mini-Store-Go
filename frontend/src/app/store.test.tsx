import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import * as api from "@/lib/api";
import type { Cart, Product, User } from "@/lib/types";
import { StoreProvider, useStore } from "./store";

vi.mock("@/lib/api", () => ({
  getCurrentUser:vi.fn(),getCart:vi.fn(),signIn:vi.fn(),signOut:vi.fn(),addCartItem:vi.fn(),createOrder:vi.fn(),
}));
const user: User={id:"real-user",name:"User",email:"user@test.invalid",role:"user",createdAt:""};
const emptyCart: Cart={items:[],itemsPrice:0,shippingPrice:0,taxPrice:0,totalPrice:0};
beforeEach(() => {
  vi.mocked(api.getCurrentUser).mockResolvedValue({success:false,message:"guest"});
  vi.mocked(api.getCart).mockResolvedValue({success:true,message:"OK",data:emptyCart});
});
afterEach(() => {cleanup();localStorage.clear();vi.resetAllMocks();});
it("ignores stored mock admin state and starts with server data", async () => {
  localStorage.setItem("mini-store-go-mock-state",JSON.stringify({currentUserId:"fake-admin",users:[{id:"fake-admin",role:"admin",password:"123456"}]}));
  const {result}=renderHook(useStore,{wrapper:StoreProvider});
  expect(result.current.currentUser).toBeNull();
  await waitFor(()=>expect(result.current.authReady).toBe(true));
  expect(result.current.state.users).toEqual([]);
  expect(result.current.state.products).toEqual([]);
  expect(localStorage.getItem("mini-store-go-mock-state")).toBeNull();
});
it("keeps sync callbacks stable and merges concurrent server batches", async () => {
  const {result}=renderHook(useStore,{wrapper:StoreProvider});
  await waitFor(()=>expect(result.current.authReady).toBe(true));
  const sync=result.current.syncProducts;
  const base={name:"Phone",slug:"phone",category:"phone",images:[],brand:"",description:"",stock:1,price:10,rating:0,numReviews:0,isFeatured:false,banner:null,createdAt:""};
  act(()=>{sync([{...base,id:"p1"} as Product]);sync([{...base,id:"p2"} as Product]);});
  expect(result.current.state.products.map(product=>product.id)).toEqual(["p1","p2"]);
  expect(result.current.syncProducts).toBe(sync);
});
it("adds to cart without requiring a locally cached product", async () => {
  vi.mocked(api.addCartItem).mockResolvedValue({success:true,message:"OK",data:emptyCart});
  const {result}=renderHook(useStore,{wrapper:StoreProvider});
  await waitFor(()=>expect(result.current.authReady).toBe(true));
  await act(async()=>expect((await result.current.addToCart("uncached-product")).success).toBe(true));
  expect(api.addCartItem).toHaveBeenCalledWith("uncached-product");
});
it("clears account state and ignores an in-flight mutation after sign out", async () => {
  vi.mocked(api.getCurrentUser).mockResolvedValue({success:true,message:"OK",data:user});
  vi.mocked(api.signOut).mockResolvedValue({success:true,message:"OK",data:{signed_out:true}});
  let finish!: (value: api.ApiResult<Cart>) => void;
  vi.mocked(api.addCartItem).mockReturnValue(new Promise(resolve=>{finish=resolve}));
  const {result}=renderHook(useStore,{wrapper:StoreProvider});
  await waitFor(()=>expect(result.current.currentUser?.id).toBe(user.id));
  const adding=result.current.addToCart("p1");
  await act(async()=>{await result.current.signOut()});
  await act(async()=>{finish({success:true,message:"OK",data:{...emptyCart,totalPrice:999}});await adding});
  expect(result.current.currentUser).toBeNull();
  expect(result.current.state.users).toEqual([]);
  expect(result.current.state.cart.totalPrice).toBe(0);
});

it("starts even when browser storage is unavailable", async () => {
  const blocked = vi.spyOn(Storage.prototype, "removeItem").mockImplementation(() => { throw new Error("Storage disabled"); });
  try {
    const {result}=renderHook(useStore,{wrapper:StoreProvider});
    await waitFor(()=>expect(result.current.authReady).toBe(true));
    expect(result.current.currentUser).toBeNull();
  } finally { blocked.mockRestore(); }
});
