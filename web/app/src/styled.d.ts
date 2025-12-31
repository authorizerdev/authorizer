import 'styled-components';

declare module 'styled-components' {
	export interface DefaultTheme {
		colors: {
			primary: string;
			primaryDisabled: string;
			gray: string;
			danger: string;
			success: string;
			textColor: string;
		};
		fonts: {
			fontStack: string;
			largeText: string;
			mediumText: string;
			smallText: string;
			tinyText: string;
		};
		radius: {
			card: string;
			button: string;
			input: string;
		};
	}
}
