const { useState, useEffect, useRef, useLayoutEffect } = React

import App from './src/App.js'

const domContainer = document.querySelector('#content');
const root = ReactDOM.createRoot(domContainer);
root.render(<App />);
