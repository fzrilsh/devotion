import HomeLayout from "@components/layout/HomeLayout";
import AboutSection from "@components/sections/home/AboutSection";
import CapacityPreviewSection from "@components/sections/home/CapacityPreviewSection";
import FinalCTASection from "@components/sections/home/FinalCTASection";
import HeroSection from "@components/sections/home/HeroSection";
import HowItWorksSection from "@components/sections/home/HowItWorksSection";
import TrustStatsSection from "@components/sections/home/TrustStatsSection";

export default function Home() {
    return (
        <HomeLayout>
            <HeroSection />
            <AboutSection />
            <HowItWorksSection />
            <CapacityPreviewSection />
            <TrustStatsSection />
            <FinalCTASection />
        </HomeLayout>
    );
}
