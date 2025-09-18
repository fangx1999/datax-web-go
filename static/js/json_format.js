// json_format.js - simple JSON formatter and minifier for the JSON tool page
// This script is loaded on the JSON 工具 page. It reads the text from
// <textarea id="jsonInput">, attempts to parse it as JSON and writes
// formatted or minified output into <pre id="jsonOutput">. Errors are
// reported via alert dialogs.

function formatJSON() {
  const inputElem = document.getElementById('jsonInput');
  const outputElem = document.getElementById('jsonOutput');
  try {
    const obj = JSON.parse(inputElem.value);
    outputElem.textContent = JSON.stringify(obj, null, 2);
  } catch (err) {
    alert('JSON 解析错误: ' + err.message);
  }
}

function minifyJSON() {
  const inputElem = document.getElementById('jsonInput');
  const outputElem = document.getElementById('jsonOutput');
  try {
    const obj = JSON.parse(inputElem.value);
    outputElem.textContent = JSON.stringify(obj);
  } catch (err) {
    alert('JSON 解析错误: ' + err.message);
  }
}