'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-css-reset.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-css-reset.cjs.dev.js");
}
