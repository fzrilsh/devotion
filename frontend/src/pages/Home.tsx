import HomeLayout from "@components/layout/HomeLayout";
import AboutSection from "@components/sections/home/AboutSection";
import CapacitySection from "@components/sections/home/CapacitySection";
// import FinalCTASection from "@components/sections/home/FinalCTASection";
import HeroSection from "@components/sections/home/HeroSection";
import HowItWorksSection from "@components/sections/home/HowItWorksSection";
// import TrustStatsSection from "@components/sections/home/TrustStatsSection";

export default function Home() {
    return (
        <HomeLayout>
            <HeroSection />
            <AboutSection />
            <HowItWorksSection />
            <CapacitySection />
            {/* <TrustStatsSection /> */}
            {/* <FinalCTASection /> */}
        </HomeLayout>
    );
}
