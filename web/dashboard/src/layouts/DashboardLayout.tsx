import React, { useState } from 'react';
import { Sidebar, MobileNav } from '../components/Sidebar';
import { Sheet, SheetContent } from '../components/ui/sheet';

export function DashboardLayout({ children }: { children: React.ReactNode }) {
	const [mobileOpen, setMobileOpen] = useState(false);
	return (
		<div className="min-h-screen bg-gray-100">
			{/* Desktop sidebar */}
			<div className="hidden md:block">
				<Sidebar onClose={() => setMobileOpen(false)} />
			</div>

			{/* Mobile sidebar via Sheet */}
			<Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
				<SheetContent side="left" className="w-64 p-0">
					<Sidebar onClose={() => setMobileOpen(false)} />
				</SheetContent>
			</Sheet>

			{/* Top nav for mobile */}
			<MobileNav onOpen={() => setMobileOpen(true)} />

			{/* Main content */}
			<div className="md:pl-64 p-4 pt-16 md:pt-4">{children}</div>
		</div>
	);
}
