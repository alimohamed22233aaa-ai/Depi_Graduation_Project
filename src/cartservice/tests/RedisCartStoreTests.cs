using System;
using System.Threading;
using System.Threading.Tasks;
using cartservice.cartstore;
using Grpc.Core;
using Microsoft.Extensions.Caching.Distributed;
using Microsoft.Extensions.Caching.Memory;
using Microsoft.Extensions.Options;
using Xunit;

namespace cartservice.tests
{
    public class RedisCartStoreTests
    {
        private static RedisCartStore CreateStore() =>
            new RedisCartStore(
                new MemoryDistributedCache(Options.Create(new MemoryDistributedCacheOptions())));

        [Fact]
        public async Task GetCart_UnknownUser_ReturnsEmptyCart()
        {
            var store = CreateStore();

            var cart = await store.GetCartAsync("unknown-user");

            Assert.NotNull(cart);
            Assert.Empty(cart.Items);
        }

        [Fact]
        public async Task AddItem_NewUser_CreatesCartWithItem()
        {
            var store = CreateStore();

            await store.AddItemAsync("user1", "PROD1", 2);
            var cart = await store.GetCartAsync("user1");

            Assert.Equal("user1", cart.UserId);
            var item = Assert.Single(cart.Items);
            Assert.Equal("PROD1", item.ProductId);
            Assert.Equal(2, item.Quantity);
        }

        [Fact]
        public async Task AddItem_SameProductTwice_SumsQuantity()
        {
            var store = CreateStore();

            await store.AddItemAsync("user1", "PROD1", 1);
            await store.AddItemAsync("user1", "PROD1", 3);
            var cart = await store.GetCartAsync("user1");

            var item = Assert.Single(cart.Items);
            Assert.Equal(4, item.Quantity);
        }

        [Fact]
        public async Task AddItem_DifferentProducts_KeepsBoth()
        {
            var store = CreateStore();

            await store.AddItemAsync("user1", "PROD1", 1);
            await store.AddItemAsync("user1", "PROD2", 5);
            var cart = await store.GetCartAsync("user1");

            Assert.Equal(2, cart.Items.Count);
        }

        [Fact]
        public async Task AddItem_DifferentUsers_AreIsolated()
        {
            var store = CreateStore();

            await store.AddItemAsync("userA", "PROD1", 1);
            var cartB = await store.GetCartAsync("userB");

            Assert.Empty(cartB.Items);
        }

        [Fact]
        public async Task EmptyCart_RemovesAllItems()
        {
            var store = CreateStore();
            await store.AddItemAsync("user1", "PROD1", 1);
            await store.AddItemAsync("user1", "PROD2", 1);

            await store.EmptyCartAsync("user1");
            var cart = await store.GetCartAsync("user1");

            Assert.Empty(cart.Items);
        }

        [Fact]
        public void Ping_ReturnsTrue()
        {
            Assert.True(CreateStore().Ping());
        }

        [Fact]
        public async Task AddItem_CacheFails_ThrowsRpcException()
        {
            var store = new RedisCartStore(new FailingCache());

            var ex = await Assert.ThrowsAsync<RpcException>(
                () => store.AddItemAsync("user1", "PROD1", 1));

            Assert.Equal(StatusCode.FailedPrecondition, ex.StatusCode);
        }

        [Fact]
        public async Task GetCart_CacheFails_ThrowsRpcException()
        {
            var store = new RedisCartStore(new FailingCache());

            var ex = await Assert.ThrowsAsync<RpcException>(
                () => store.GetCartAsync("user1"));

            Assert.Equal(StatusCode.FailedPrecondition, ex.StatusCode);
        }

        [Fact]
        public async Task EmptyCart_CacheFails_ThrowsRpcException()
        {
            var store = new RedisCartStore(new FailingCache());

            var ex = await Assert.ThrowsAsync<RpcException>(
                () => store.EmptyCartAsync("user1"));

            Assert.Equal(StatusCode.FailedPrecondition, ex.StatusCode);
        }

        // Fake cache that always throws, to exercise the error-handling paths.
        private sealed class FailingCache : IDistributedCache
        {
            public byte[] Get(string key) => throw new InvalidOperationException("cache down");
            public Task<byte[]> GetAsync(string key, CancellationToken token = default) =>
                throw new InvalidOperationException("cache down");
            public void Set(string key, byte[] value, DistributedCacheEntryOptions options) =>
                throw new InvalidOperationException("cache down");
            public Task SetAsync(string key, byte[] value, DistributedCacheEntryOptions options,
                CancellationToken token = default) =>
                throw new InvalidOperationException("cache down");
            public void Refresh(string key) { }
            public Task RefreshAsync(string key, CancellationToken token = default) => Task.CompletedTask;
            public void Remove(string key) { }
            public Task RemoveAsync(string key, CancellationToken token = default) => Task.CompletedTask;
        }
    }
}
