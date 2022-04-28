import React, { lazy, Suspense } from "react";
import { Outlet, Route, Routes } from "react-router-dom";

import { useAuthContext } from "../contexts/AuthContext";
import { DashboardLayout } from "../layouts/DashboardLayout";

// @Component PAGES
const Auth = lazy(() => import("../pages/Auth"));
const Environment = lazy(() => import("../pages/Environment"));
const Home = lazy(() => import("../pages/Home"));
const Users = lazy(() => import("../pages/Users"));

export const AppRoutes = () => {
  const { isLoggedIn } = useAuthContext();

  if (isLoggedIn) {
    return (
      <Suspense fallback={<></>}>
        <Routes>
          <Route
            element={
              <DashboardLayout>
                <Outlet />
              </DashboardLayout>
            }
          >
            {/* <Route path="/" element={<Environment />} /> */}
            <Route path="/" element={<Outlet />}>
              <Route index element={<Environment />} />
              <Route path="/:sec" element={<Environment />} />
            </Route>
            <Route path="users" element={<Users />} />
            <Route path="*" element={<Home />} />
          </Route>
        </Routes>
      </Suspense>
    );
  }
  // <Route path="/environment" element={<Outlet />}>
  //   <Route index={false} path="/:sec" element={<Environment />} />
  // </Route>;
  return (
    <Suspense fallback={<></>}>
      <Routes>
        <Route path="/" element={<Auth />} />
        <Route path="*" element={<Auth />} />
      </Routes>
    </Suspense>
  );
};
