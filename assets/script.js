const input = document.getElementById('input')
const output = document.getElementById('output')

if (!WebAssembly.instantiateStreaming) { // polyfill
    WebAssembly.instantiateStreaming = async (resp, importObject) => {
        const source = await (await resp).arrayBuffer();
        return await WebAssembly.instantiate(source, importObject);
    };
}

const go = new Go();
let mod, inst;
WebAssembly.instantiateStreaming(fetch("json.wasm"), go.importObject).then((result) => {
    mod = result.module;
    inst = result.instance;
    gorun();
}).catch((err) => {
    console.error(err);
});

async function gorun() {
    await go.run(inst);
    inst = await WebAssembly.instantiate(mod, go.importObject); // reset instance
}

input.onchange = function (event) {
  const files = event.target.files

  output.innerHTML = ''
  
  for(let file of files) {
    const reader = new FileReader()

    reader.readAsDataURL(file)
    reader.onload = function fileReadCompleted() {
        const image = new Image()
        image.src = reader.result
        output.appendChild(image)

        const regex = /data:.*;base64,/i;
        res = phashGo(reader.result.replace(regex, ''))
        console.log("js res: ",res)

        const textbox = document.createElement('div')
        textbox.innerText = `phash: ${res}`
        output.appendChild(textbox)
    };
  }

}