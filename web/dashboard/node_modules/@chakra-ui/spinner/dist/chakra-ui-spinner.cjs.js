'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-spinner.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-spinner.cjs.dev.js");
}
