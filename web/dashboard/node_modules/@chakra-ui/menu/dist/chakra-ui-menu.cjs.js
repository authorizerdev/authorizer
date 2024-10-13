'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-menu.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-menu.cjs.dev.js");
}
