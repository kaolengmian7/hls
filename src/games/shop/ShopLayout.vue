<template>
  <div class="shop-bg flex-1 flex flex-col">
    <router-view />
  </div>
</template>

<style scoped>
.shop-bg {
  background-image:
    linear-gradient(rgba(10, 5, 2, 0.45), rgba(10, 5, 2, 0.45)),
    url('/shop-bg.webp');
  background-size: cover;
  background-position: center;
}
</style>
