// YogaLoad instantiates a Go WASM Yoga app and exposes linear memory as
// globalThis.wasm.instance.exports.mem for cogentcore/webgpu jsx.BytesToJS.
async function YogaLoad(opts) {
  opts = opts || {};
  var wasmURL = opts.wasmURL || "app.wasm";

  if (!navigator.gpu) {
    throw new Error("WebGPU is not available in this browser (navigator.gpu missing)");
  }

  if (typeof Go !== "function") {
    throw new Error("wasm_exec.js did not define Go");
  }

  var go = new Go();
  var result;
  if (typeof WebAssembly.instantiateStreaming === "function") {
    try {
      result = await WebAssembly.instantiateStreaming(fetch(wasmURL), go.importObject);
    } catch (e) {
      // Fall back when MIME type is wrong (some static servers).
      var resp = await fetch(wasmURL);
      result = await WebAssembly.instantiate(await resp.arrayBuffer(), go.importObject);
    }
  } else {
    var response = await fetch(wasmURL);
    result = await WebAssembly.instantiate(await response.arrayBuffer(), go.importObject);
  }

  // Required by github.com/cogentcore/webgpu/jsx.BytesToJS
  globalThis.wasm = { instance: result.instance };

  var boot = document.getElementById("yoga-boot");
  if (boot) {
    boot.remove();
  }

  go.run(result.instance);
  return result.instance;
}
