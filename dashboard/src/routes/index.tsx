import React from "react";
import { Outlet, Route, Routes } from "react-router-dom";
import { DashboardLayout } from "../layouts/DashboardLayout";
import { Auth } from "../pages/Auth";

import { Home } from "../pages/Home";
import { Users } from "../pages/Users";

export const AppRoutes = () => {
  return (
    <Routes>
      <Route path="login" element={<Auth />} />
      <Route path="setup" element={<Auth />} />
      <Route
        element={
          <DashboardLayout>
            <Outlet />
          </DashboardLayout>
        }
      >
        <Route path="/" element={<Home />} />
        <Route path="users" element={<Users />} />
      </Route>
    </Routes>
  );
};
