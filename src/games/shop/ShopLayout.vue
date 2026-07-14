<template>
  <div class="shop-font-sans flex-1 flex flex-col">
    <router-view />
  </div>
</template>

<style scoped>
.shop-font-sans {
  font-family: "Noto Sans SC", "PingFang SC", "Microsoft YaHei", sans-serif;
}

.shop-font-sans :deep(.font-pixel),
.shop-font-sans :deep(.coin-tag),
.shop-font-sans :deep(h1),
.shop-font-sans :deep(h2),
.shop-font-sans :deep(h3),
.shop-font-sans :deep(h4),
.shop-font-sans :deep(h5),
.shop-font-sans :deep(.hls-panel-title),
.shop-font-sans :deep(.hls-panel-title-light) {
  font-family: inherit;
}
</style>
