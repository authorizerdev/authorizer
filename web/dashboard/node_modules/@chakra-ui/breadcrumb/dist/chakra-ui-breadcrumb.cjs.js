'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-breadcrumb.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-breadcrumb.cjs.dev.js");
}
