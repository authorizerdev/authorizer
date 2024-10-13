'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-slider.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-slider.cjs.dev.js");
}
