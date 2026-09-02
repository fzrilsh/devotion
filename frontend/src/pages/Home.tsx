import HomeLayout from "@components/layout/HomeLayout";
import AboutSection from "@components/sections/home/AboutSection";
import CapacitySection from "@components/sections/home/CapacitySection";
import HeroSection from "@components/sections/home/HeroSection";
import HowItWorksSection from "@components/sections/home/HowItWorksSection";

export default function Home() {
    return (
        <HomeLayout>
            <HeroSection />
            <AboutSection />
            <HowItWorksSection />
            <CapacitySection />
        </HomeLayout>
    );
}
